package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/rancher/websocket-proxy/proxy/apiinterceptor/model"
	check "gopkg.in/check.v1"
	"os"
)

func Test(t *testing.T) { check.TestingT(t) }

type InterceptorTestSuite struct {
	mockCattle      *mockCattleServer
	mockInterceptor *mockInterceptor
}

var _ = check.Suite(&InterceptorTestSuite{})

func (i *InterceptorTestSuite) SetUpSuite(c *check.C) {
	conf := getInterceptorTestConfig()
	interceptorConf := `
	{
	"requestInterceptors": [
		{
			"type": "authTokenValidator",
			"paths": ["/interceptor"],
			"methods": ["get", "post", "delete"]
		},
		{
			"type": "http",
			"paths": ["/v1/services", "/v1/services/{id}"],
			"endpoint": "http://localhost:5552/interceptor",
			"methods": ["post"],
			"secretToken": ""
		}
	],
	"destinations": [
		{
			"paths": ["/external-service"],
			"destinationURL": "http://localhost:5553"
		}
	]
	}
	`
	// Current config code is coupled to reading from file
	if err := ioutil.WriteFile(conf.APIInterceptorConfigFile, []byte(interceptorConf), 0644); err != nil {
		c.Fatal("Failed to write config file", err)
	}

	ps := &Starter{
		CattleProxyPaths: []string{"/{cattle-proxy:.*}"},
		Config:           conf,
	}
	go ps.StartProxy()

	// This mock will receive and store requests that would have gone to cattle
	cattleRouter := mux.NewRouter()
	mc := &mockCattleServer{c}
	cattleRouter.Handle("/{cattle:.*}", mc)
	go http.ListenAndServe("127.0.0.1:5551", cattleRouter)

	// This mock will receive and store requests that would have gone to cattle
	interceptorRouter := mux.NewRouter()
	mi := &mockInterceptor{c: c}
	interceptorRouter.Handle("/interceptor", mi)
	go http.ListenAndServe("127.0.0.1:5552", interceptorRouter)

	i.mockCattle = mc
	i.mockInterceptor = mi

	// Allow servers time to initialize
	time.Sleep(50 * time.Millisecond)

	// Reload the config
	if _, err := http.Post("http://localhost:5550/v1-api-interceptor/reload", "aplication/json", nil); err != nil {
		c.Fatal("Error reloading interceptor config", err)
	}
}

func (i *InterceptorTestSuite) TearDownSuite(c *check.C) {
	conf := getInterceptorTestConfig()
	os.Remove(conf.APIInterceptorConfigFile)
}

func (i *InterceptorTestSuite) TestInterceptor(c *check.C) {
	// Construct the original client request
	origHeaders := map[string][]string{
		"Header1":      {"header-val-1"},
		"Content-Type": {"application/json"},
	}
	originalBody := map[string]interface{}{
		"name":   "name1",
		"field1": "value1",
	}
	b := new(bytes.Buffer)
	if err := json.NewEncoder(b).Encode(originalBody); err != nil {
		c.Fatal("Couldn't encode request body", err)
	}
	req, err := http.NewRequest("POST", "http://localhost:5550/v1/services", b)
	if err != nil {
		c.Fatal("Couldn't create request", err)
	}
	req.Header.Add("header1", origHeaders["Header1"][0])
	req.Header.Add("Content-Type", origHeaders["Content-Type"][0])

	// This is the response the mockInterceptor will send back
	// It modifies the name field, adds newField, and adds a header
	respHeaders := map[string][]string{}
	for k, v := range origHeaders {
		respHeaders[k] = v
	}
	respHeaders["NewHeader"] = []string{"new-header-val"}
	respBody := map[string]interface{}{}
	for k, v := range originalBody {
		respBody[k] = v
	}
	respBody["name"] = "new-name"
	respBody["newField"] = "new-field"
	intResp := model.APIRequestData{
		Headers: respHeaders,
		Body:    respBody,
	}
	i.mockInterceptor.response = intResp

	// Perform the client request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.Fatal("Couldn't read response: ", err)
	}
	// Verify the response from cattle. Since the mock cattle server just echos the request it received,
	// the response should match the modified response.
	cattleRespBody := map[string]interface{}{}
	if err = json.NewDecoder(resp.Body).Decode(&cattleRespBody); err != nil {
		c.Fatal("Coudln't decode mock cattle response: ", err)
	}
	c.Check(cattleRespBody, check.DeepEquals, respBody)
	for k, vals := range respHeaders {
		actualVals := resp.Header[http.CanonicalHeaderKey(k)]
		c.Assert(actualVals, check.DeepEquals, vals)
	}

	// Verify the request that came into the interceptor
	var actualReq model.APIRequestData
	if err := json.Unmarshal(i.mockInterceptor.actualRequestBody, &actualReq); err != nil {
		c.Fatal("Couldn't decode actual request body: ", err)
	}
	// Some irrelevant headers are added by the go client, ignore these
	for k := range actualReq.Headers {
		if k != "Content-Type" && k != "Header1" {
			delete(actualReq.Headers, k)
		}
	}
	c.Assert(actualReq.Headers, check.DeepEquals, origHeaders)
	c.Assert(actualReq.Body, check.DeepEquals, originalBody)
	c.Assert(actualReq.UUID, check.Not(check.Equals), "")
	c.Assert(actualReq.APIPath, check.Equals, "/v1/services")
	c.Assert(actualReq.APIMethod, check.Equals, "POST")
}

func (i *InterceptorTestSuite) TestInterceptorError(c *check.C) {
	// Construct the original client request

	origHeaders := map[string][]string{
		"Header1":      {"header-val-1"},
		"Content-Type": {"application/json"},
	}
	originalBody := map[string]interface{}{
		"name":   "name1",
		"field1": "value1",
	}
	b := new(bytes.Buffer)
	if err := json.NewEncoder(b).Encode(originalBody); err != nil {
		c.Fatal("Couldn't encode request body", err)
	}
	req, err := http.NewRequest("POST", "http://localhost:5550/v1/services", b)
	if err != nil {
		c.Fatal("Couldn't create request", err)
	}
	req.Header.Add("header1", origHeaders["Header1"][0])
	req.Header.Add("Content-Type", origHeaders["Content-Type"][0])

	// This is the response the mockInterceptor will send back
	// It returns a 400 error code and a message

	intResp := model.APIRequestData{
		Message: "Bad Request error from Mock Interceptor",
	}
	i.mockInterceptor.response = intResp
	i.mockInterceptor.responseStatus = 400

	// Perform the client request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.Fatal("Couldn't read response: ", err)
	}
	// Verify the response from cattle. Since the mock cattle server just echos the request it received,
	// the response should match the modified response.
	cattleRespBody := map[string]interface{}{}
	if err = json.NewDecoder(resp.Body).Decode(&cattleRespBody); err != nil {
		c.Fatal("Coudln't decode mock cattle response: ", err)
	}
	c.Assert(cattleRespBody["status"], check.Equals, "400")
	c.Assert(cattleRespBody["message"], check.Equals, "Bad Request error from Mock Interceptor")
}

type mockInterceptor struct {
	actualRequest     *http.Request
	actualRequestBody []byte
	response          interface{}
	responseStatus    int
	c                 *check.C
}

func (m *mockInterceptor) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	var err error
	m.actualRequestBody, err = ioutil.ReadAll(req.Body)
	if err != nil {
		m.c.Fatal("Error reading body", err)
	}
	m.actualRequest = req
	if m.responseStatus != 0 {
		rw.WriteHeader(m.responseStatus)
	}
	json.NewEncoder(rw).Encode(m.response)
}

type mockCattleServer struct {
	c *check.C
}

func (m *mockCattleServer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		m.c.Fatal("Error reading body", err)
	}

	for k, vals := range req.Header {
		for _, v := range vals {
			rw.Header().Add(k, v)
		}
	}

	fmt.Fprint(rw, string(body))
}

func getInterceptorTestConfig() *Config {
	ports := map[int]bool{443: true}
	config := &Config{
		ListenAddr:               "127.0.0.1:5550",
		CattleAddr:               "127.0.0.1:5551",
		ProxyProtoHTTPSPorts:     ports,
		APIInterceptorConfigFile: "test-config.json",
	}
	return config
}
