package apiinterceptor

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/rancher/websocket-proxy/proxy/apiinterceptor/filters"
	"github.com/rancher/websocket-proxy/proxy/apiinterceptor/filters/auth"
	httpfilter "github.com/rancher/websocket-proxy/proxy/apiinterceptor/filters/http"
	"github.com/rancher/websocket-proxy/proxy/apiinterceptor/model"
)

//destination defines the properties of a destination
type destination struct {
	DestinationURL string   `json:"DestinationURL"`
	Paths          []string `json:"Paths"`
}

//configFileFields stores filter config
type configFileFields struct {
	RequestInterceptors []model.FilterData
	Destinations        []destination
}

func newRouter(configFile string, cattleAddr string, routerSetter routerSetter) (http.Handler, error) {
	if cattleAddr == "" {
		return nil, fmt.Errorf("No CattleAddr set in proxy config to forward the requests to Cattle")
	}

	cattleURL := "http://" + cattleAddr
	url, err := url.Parse(cattleURL)
	if err != nil {
		return nil, errors.Wrapf(err, "Couldn't parse cattle url %v", url)
	}

	director := func(req *http.Request) {
		req.URL.Scheme = "http"
		req.URL.Host = cattleAddr
	}
	cattleRevProxy := &httputil.ReverseProxy{
		Director:      director,
		FlushInterval: time.Millisecond * 100,
	}

	apiFilters, err := loadAPIFilters()
	if err != nil {
		return nil, errors.Wrap(err, "Couldn't load API filters")
	}
	return buildRouter(configFile, cattleRevProxy, apiFilters, routerSetter)
}

func buildRouter(configFile string, cattleRevProxy *httputil.ReverseProxy, apiFilters map[string]filters.APIFilter, routerSetter routerSetter) (http.Handler, error) {
	pathPreFilters := map[string][]model.FilterData{}
	pathDestinations := map[string]http.Handler{}
	configFields := configFileFields{}

	if configFile != "" {
		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			//file does not exist, treating it as empty config, since cattle deletes the file when config is set to empty
			log.Debugf("config.json file not found %v", configFile)
		} else {
			configContent, err := ioutil.ReadFile(configFile)
			if err != nil {
				return nil, errors.Wrapf(err, "Error reading config.json file at path %v", configFile)
			}

			err = json.Unmarshal(configContent, &configFields)
			if err != nil {
				return nil, errors.Wrap(err, "Couldn't unmarshal config.json")
			}

			for _, filter := range configFields.RequestInterceptors {
				for _, path := range filter.Paths {
					pathPreFilters[path] = append(pathPreFilters[path], filter)
				}
			}

			for _, destination := range configFields.Destinations {
				//build the pathDestinations map
				destProxy, err := newProxy(destination.DestinationURL)
				if err != nil {
					return nil, errors.Wrapf(err, "Couldn't load proxy for destination %v", destination)
				}
				for _, path := range destination.Paths {
					pathDestinations[path] = destProxy
				}
			}
		}
	}

	copyAPIFilters := map[string]filters.APIFilter{}
	for k, v := range apiFilters {
		copyAPIFilters[k] = v
	}

	interceptor := &interceptor{
		configFile:         configFile,
		cattleReverseProxy: cattleRevProxy,
		apiFilters:         copyAPIFilters,
		pathDestinations:   pathDestinations,
		pathPreFilters:     pathPreFilters,
		routerSetter:       routerSetter,
	}

	router := mux.NewRouter().StrictSlash(false)
	for _, filter := range configFields.RequestInterceptors {
		//build interceptor Paths
		for _, path := range filter.Paths {
			for _, method := range filter.Methods {
				log.Infof("Adding route: %v %v", strings.ToUpper(method), path)
				router.Methods(strings.ToUpper(method)).Path(path).HandlerFunc(http.HandlerFunc(interceptor.intercept))
			}
		}
	}

	router.Methods("POST").Path("/v1-api-interceptor/reload").HandlerFunc(http.HandlerFunc(interceptor.reload))
	router.NotFoundHandler = http.HandlerFunc(interceptor.cattleProxy)
	var routes []*mux.Route
	router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		routes = append(routes, route)
		return nil
	})
	interceptor.routes = routes
	return router, nil
}

func loadAPIFilters() (map[string]filters.APIFilter, error) {
	apiFilters := make(map[string]filters.APIFilter)

	httpFilter, err := httpfilter.NewFilter()
	if err != nil {
		return nil, errors.Wrapf(err, "Couldn't initalize APIFilter %v", httpFilter.GetType())
	}
	apiFilters[httpFilter.GetType()] = httpFilter

	tokenFilter, err := auth.NewFilter()
	if err != nil {
		return nil, errors.Wrapf(err, "Couldn't initalize APIFilter %v", httpFilter.GetType())
	}
	apiFilters[tokenFilter.GetType()] = tokenFilter

	return apiFilters, nil
}

func newProxy(target string) (*httputil.ReverseProxy, error) {
	url, err := url.Parse(target)
	if err != nil {
		log.Errorf("Error reading destination URL %v", target)
		return nil, err
	}
	newProxy := httputil.NewSingleHostReverseProxy(url)
	newProxy.FlushInterval = time.Millisecond * 100
	return newProxy, nil
}
