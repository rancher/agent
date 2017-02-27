package apiinterceptor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"strconv"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/pborman/uuid"
	"github.com/rancher/websocket-proxy/proxy/apiinterceptor/filters"
	"github.com/rancher/websocket-proxy/proxy/apiinterceptor/model"
)

type interceptor struct {
	configFile         string
	routerSetter       routerSetter
	cattleReverseProxy *httputil.ReverseProxy
	apiFilters         map[string]filters.APIFilter
	pathPreFilters     map[string][]model.FilterData
	pathDestinations   map[string]http.Handler
	routes             []*mux.Route
}

func (i *interceptor) intercept(w http.ResponseWriter, req *http.Request) {
	path, _ := mux.CurrentRoute(req).GetPathTemplate()

	var otherPathsMatched []string
	for _, route := range i.routes {
		var match mux.RouteMatch
		if route.Match(req, &match) {
			routePath, err := route.GetPathTemplate()
			if err == nil && routePath != path && !containsPath(otherPathsMatched, routePath) {
				otherPathsMatched = append(otherPathsMatched, routePath)
			}
		}
	}

	logrus.Debugf("Request Path matched: %v ,Other Matching paths: %v", path, otherPathsMatched)

	bodyBytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		logrus.Errorf("Error reading request Body %v for path %v", req, path)
		returnHTTPError(w, req, http.StatusBadRequest, fmt.Sprintf("Error reading json request body, err: %v", err))
		return
	}

	var jsonInput map[string]interface{}
	if len(bodyBytes) > 0 {
		err = json.Unmarshal(bodyBytes, &jsonInput)
		if err != nil {
			logrus.Errorf("Error unmarshalling json request body: %v", err)
			returnHTTPError(w, req, http.StatusBadRequest, fmt.Sprintf("Error reading json request body: %v", err))
			return
		}
	}

	headerMap := make(map[string][]string)
	for key, value := range req.Header {
		headerMap[key] = value
	}

	api := req.URL.Path
	method := req.Method

	inputBody, inputHeaders, destination, proxyErr := i.processPreFilters(path, otherPathsMatched, api, method, jsonInput, headerMap)
	if proxyErr.Status != "" {
		//error from some filter
		logrus.Debugf("Error processing request interceptor %v", proxyErr)
		writeError(w, proxyErr)
		return
	}

	jsonStr, err := json.Marshal(inputBody)
	req.Body = ioutil.NopCloser(bytes.NewReader(jsonStr))
	req.ContentLength = int64(len(jsonStr))

	// In future, consider changing behavior to clear out all headers on request, based on user feedback
	for key, value := range inputHeaders {
		req.Header[http.CanonicalHeaderKey(key)] = value
	}

	destination.ServeHTTP(w, req)
}

func (i *interceptor) processPreFilters(path string, otherPathsMatched []string, api string, method string, body map[string]interface{}, headers map[string][]string) (map[string]interface{}, map[string][]string, http.Handler, model.ProxyError) {
	destinationProxy, ok := i.pathDestinations[path]
	if !ok {
		destinationProxy = i.cattleReverseProxy
	}

	logrus.Debugf("START -- Processing requestInterceptors for request path %v method %v", api, method)
	inputBody := body
	inputHeaders := headers
	//add uuid
	UUID := generateUUID()
	//envId
	envID := extractEnvID(api)

	preFilters := i.pathPreFilters[path]

	for _, pathMatched := range otherPathsMatched {
		for _, otherFilter := range i.pathPreFilters[pathMatched] {
			//append if not duplicate
			if !containsFilter(preFilters, otherFilter) {
				preFilters = append(preFilters, otherFilter)
			}
		}
	}

	for _, filterData := range preFilters {
		var methodMatch bool
		methodMatch = false
		for _, m := range filterData.Methods {
			if strings.ToUpper(m) == method {
				methodMatch = true
				break
			}
		}

		if !methodMatch {
			continue
		}

		logrus.Debugf("-- Processing requestInterceptor %v for request path %v --", filterData, api)

		requestData := model.APIRequestData{}
		requestData.Body = inputBody
		requestData.Headers = inputHeaders
		requestData.UUID = UUID
		requestData.APIPath = api
		requestData.APIMethod = method
		if envID != "" {
			requestData.EnvID = envID
		}

		apiFilter, ok := i.apiFilters[filterData.Type]
		if !ok {
			logrus.Errorf("Skipping interceptor type %v doesn't exist.", filterData.Type)
			continue
		}

		responseData, err := apiFilter.ProcessFilter(filterData, requestData)
		if err != nil {
			logrus.Errorf("Error %v processing the interceptor %v", err, filterData)
			svcErr := model.ProxyError{
				Status:  strconv.Itoa(http.StatusInternalServerError),
				Message: fmt.Sprintf("Error %v processing the interceptor %v", err, filterData),
			}
			return inputBody, inputHeaders, nil, svcErr
		}
		if responseData.Status == 200 {
			if responseData.Body != nil {
				inputBody = responseData.Body
			}
			if responseData.Headers != nil {
				inputHeaders = responseData.Headers
			}
		} else {
			//error
			logrus.Errorf("Error response %v - %v while processing the interceptor %v for request path %v", responseData.Status, responseData.Message, filterData, api)
			message := fmt.Sprintf("Error response while processing the interceptor endpoint %v", filterData.Endpoint)
			if responseData.Message != "" {
				message = responseData.Message
			}

			svcErr := model.ProxyError{
				Status:  strconv.Itoa(responseData.Status),
				Message: message,
			}

			return inputBody, inputHeaders, nil, svcErr
		}
	}

	//send the final body and headers to destination

	logrus.Debugf("DONE -- Processing requestInterceptors for request path %v", api)

	return inputBody, inputHeaders, destinationProxy, model.ProxyError{}
}

func (i *interceptor) cattleProxy(w http.ResponseWriter, req *http.Request) {
	i.cattleReverseProxy.ServeHTTP(w, req)
}

func (i *interceptor) reload(w http.ResponseWriter, req *http.Request) {
	logrus.Info("reload proxy config")
	router, err := buildRouter(i.configFile, i.cattleReverseProxy, i.apiFilters, i.routerSetter)
	if err != nil {
		//failed to reload the config from the config.json
		logrus.Errorf("reload proxy config failed with error %v", err)
		returnHTTPError(w, req, http.StatusInternalServerError, fmt.Sprintf("Failed to reload the proxy config with error %v", err))
		return
	}
	i.routerSetter.setRouter(router)
}

func returnHTTPError(w http.ResponseWriter, r *http.Request, httpStatus int, errorMessage string) {
	svcError := model.ProxyError{
		Status:  strconv.Itoa(httpStatus),
		Message: errorMessage,
	}
	writeError(w, svcError)
}

func writeError(w http.ResponseWriter, svcError model.ProxyError) {
	status, err := strconv.Atoi(svcError.Status)
	if err != nil {
		logrus.Errorf("Error writing error response %v", err)
		w.Write([]byte(svcError.Message))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	jsonStr, err := json.Marshal(svcError)
	if err != nil {
		logrus.Errorf("Error writing error response %v", err)
		w.Write([]byte(svcError.Message))
		return
	}
	w.Write([]byte(jsonStr))
}

func extractEnvID(requestURL string) string {
	envID := ""
	if strings.Contains(requestURL, "/projects/") {
		parts := strings.Split(requestURL, "/projects/")
		if len(parts) > 1 {
			subParts := strings.Split(parts[1], "/")
			envID = subParts[0]
		}
	}
	return envID
}

func generateUUID() string {
	newUUID := uuid.NewUUID()
	logrus.Debugf("uuid generated: %v", newUUID)
	time, _ := newUUID.Time()
	logrus.Debugf("time generated: %v", time)
	return newUUID.String()
}

func containsPath(strs []string, newStr string) bool {
	for _, a := range strs {
		if a == newStr {
			return true
		}
	}
	return false
}

func containsFilter(filters []model.FilterData, newfilter model.FilterData) bool {
	for _, a := range filters {
		if a.Type == newfilter.Type && a.Endpoint == newfilter.Endpoint && a.SecretToken == newfilter.SecretToken {
			return true
		}
	}
	return false
}
