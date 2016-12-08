package proxy

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/rancher/websocket-proxy/common"

	. "gopkg.in/check.v1"
)

const host = "localhost:23425"

func Test(t *testing.T) {
	TestingT(t)
}

type ProxyTestSuite struct {
}

var _ = Suite(&ProxyTestSuite{})

func (s *ProxyTestSuite) TestPost(c *C) {
	input := make(chan string)
	output := make(chan common.Message)

	handler := &Handler{}
	go handler.Handle("key", "init", input, output)

	input <- marshal(c, common.HTTPMessage{
		Method: "GET",
		URL:    "http://" + host + "/foo",
		Body:   []byte("foo"),
	})
	input <- marshal(c, common.HTTPMessage{
		Body: []byte("bar"),
	})
	input <- marshal(c, common.HTTPMessage{
		EOF: true,
	})

	var response common.HTTPMessage
	unmarshal(c, <-output, &response)
	c.Assert(response.Code, Equals, 200)
	response = common.HTTPMessage{}

	//Second message will have the payload
	unmarshal(c, <-output, &response)
	c.Assert(string(response.Body), Equals, "foobar")
}

func unmarshal(c *C, msg common.Message, httpMessage *common.HTTPMessage) {
	if err := json.Unmarshal([]byte(msg.Body), httpMessage); err != nil {
		c.Fatal(err)
	}
}

func marshal(c *C, obj interface{}) string {
	bytes, err := json.Marshal(obj)
	if err != nil {
		c.Fatal(err)
	}
	return string(bytes)
}

func (s *ProxyTestSuite) SetUpSuite(c *C) {
	go http.ListenAndServe(host, http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		bytes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		rw.Write(bytes)
	}))
}
