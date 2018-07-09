package dockersocketproxy

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/gorilla/websocket"
	"github.com/rancher/agent/service/hostapi/config"
	"github.com/rancher/agent/service/hostapi/events"
	"github.com/rancher/agent/service/hostapi/testutils"
	"github.com/rancher/log"
	"github.com/rancher/websocket-proxy/backend"
	"github.com/rancher/websocket-proxy/proxy"
	wsp_utils "github.com/rancher/websocket-proxy/testutils"
	"golang.org/x/net/context"
	"gopkg.in/check.v1"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

type ProxyTestSuite struct {
	client     *client.Client
	privateKey interface{}
}

var _ = check.Suite(&ProxyTestSuite{})

func (s *ProxyTestSuite) TestSimpleCalls(c *check.C) {
	ws := s.connect(c)
	defer ws.Close()

	encoded := encodeRequest("GET", "/containers/json?all=1", nil, c)
	ws.WriteMessage(websocket.TextMessage, encoded)

	checkResponse(map[string]string{"HTTP/1.1 200 OK": ""}, ws, c)

	encoded = encodeRequest("GET", "/images/json", nil, c)
	ws.WriteMessage(websocket.TextMessage, encoded)
	checkResponse(map[string]string{"HTTP/1.1 200 OK": ""}, ws, c)
}

func (s *ProxyTestSuite) TestStartAndConnect(c *check.C) {
	ws := s.connect(c)
	defer ws.Close()

	createConfig := container.Config{
		Image:     "ibuildthecloud/helloworld:latest",
		Tty:       true,
		OpenStdin: true,
	}

	container := s.createAndStart(ws, createConfig, c)

	encoded := encodeRequest("POST", fmt.Sprintf("/containers/%s/attach?logs=1&stream=1&stdout=1", container.ID), nil, c)
	ws.WriteMessage(websocket.TextMessage, encoded)
	checkResponse(map[string]string{"Content-Type: application/vnd.docker.raw-stream": "", "Sleeping 1": "", "Sleeping 3": ""}, ws, c)
}

func (s *ProxyTestSuite) TestInteractive(c *check.C) {
	ws := s.connect(c)
	defer ws.Close()

	createConfig := container.Config{
		Image:     "ibuildthecloud/helloworld:latest",
		Tty:       true,
		OpenStdin: true,
		Cmd:       []string{"/bin/sh"},
	}

	container := s.createAndStart(ws, createConfig, c)

	encoded := encodeRequest("POST", fmt.Sprintf("/containers/%s/attach?stream=1&stdout=1&stdin=1&stderr=1", container.ID), nil, c)
	ws.WriteMessage(websocket.TextMessage, encoded)
	checkResponse(map[string]string{"Content-Type: application/vnd.docker.raw-stream": ""}, ws, c)

	encoded = encodeMessage("touch foo\n")
	ws.WriteMessage(websocket.TextMessage, encoded)
	encoded = encodeMessage("ls\n")
	ws.WriteMessage(websocket.TextMessage, encoded)
	checkResponse(map[string]string{"bin": "", "foo": ""}, ws, c)
}

/*
func (s *ProxyTestSuite) TestToCompareDockerClientBehavior(c *check.C) {

	// var reader bytes.Buffer
	// var writer bytes.Buffer
	inputReader, inputWriter := io.Pipe()
	reader, writer := io.Pipe()
	opts := docker.AttachToContainerOptions{
		RawTerminal:  true,
		Stdin:        true,
		Stdout:       true,
		Stderr:       true,
		Stream:       true,
		InputStream:  inputReader,
		OutputStream: writer,
		ErrorStream:  writer,
		Container:    "id goes here",
	}
	go s.client.AttachToContainer(opts)
	go func() {
		for {
			buff := make([]byte, 32)
			n, err := reader.Read(buff[:])
			if n > 0 {
				log.Info("I GOT THIS MSG: ", string(buff))
			}
			if err != nil {
				if err != io.EOF {
					c.Fatal(err)
				}
				return
			}
		}
	}()
	inputWriter.Write([]byte("ls\n"))
	time.Sleep(time.Second * 3)
}
*/

func (s *ProxyTestSuite) createAndStart(ws *websocket.Conn, createConfig container.Config, c *check.C) *types.ContainerJSON {
	body, err := json.Marshal(createConfig)
	if err != nil {
		c.Fatal("Failed to marshal json. %#v", err)
	}
	encoded := encodeRequest("POST", "/containers/create", body, c)
	ws.WriteMessage(websocket.TextMessage, encoded)
	respMsg := checkResponse(map[string]string{"HTTP/1.1 201 Created": ""}, ws, c)
	container := &types.ContainerJSON{}
	found := false
	for _, line := range strings.Split(respMsg, "\n") {
		if strings.HasPrefix(line, "{") {
			json.Unmarshal([]byte(line), container)
			found = true
		}
	}
	if !found {
		c.Fatal("Didn't find body!")
	}
	encoded = encodeRequest("POST", fmt.Sprintf("/containers/%s/start", container.ID), nil, c)
	ws.WriteMessage(websocket.TextMessage, encoded)
	respMsg = checkResponse(map[string]string{"HTTP/1.1 204 No Content": ""}, ws, c)
	return container
}

func (s *ProxyTestSuite) connect(c *check.C) *websocket.Conn {
	dialer := &websocket.Dialer{}
	headers := http.Header{}
	payload := map[string]interface{}{
		"hostUuid": "1",
	}
	token := wsp_utils.CreateTokenWithPayload(payload, s.privateKey)
	url := "ws://localhost:4444/v1/dockersocket/?token=" + token
	ws, _, err := dialer.Dial(url, headers)
	if err != nil {
		c.Fatal(err)
	}
	return ws
}

func checkResponse(checkFor map[string]string, ws *websocket.Conn, c *check.C) string {
	for count := 0; count < 20; count++ {
		_, m, err := ws.ReadMessage()
		if err != nil {
			// ws closed
			if len(checkFor) != 0 {
				c.Fatal("Didn't find all keys before ws was closed: ", checkFor)
			}
			return ""
		}
		dst := make([]byte, base64.StdEncoding.EncodedLen(len(m)))
		_, err = base64.StdEncoding.Decode(dst, m)
		if err != nil {
			c.Fatal(err)
		}
		msg := string(dst)
		for k := range checkFor {
			if strings.Contains(msg, k) {
				delete(checkFor, k)
			}
		}
		if len(checkFor) == 0 {
			return msg
		}
	}

	if len(checkFor) != 0 {
		c.Fatal("Didn't find: ", checkFor)
	}

	return ""
}

func encodeMessage(msg string) []byte {
	encoded := make([]byte, base64.StdEncoding.EncodedLen(len(msg)))
	base64.StdEncoding.Encode(encoded, []byte(msg))
	return encoded
}

func encodeRequest(method string, uri string, body []byte, c *check.C) []byte {
	reader := bytes.NewReader(body)
	req, err := http.NewRequest(method, "http://foo"+uri, reader)
	if err != nil {
		c.Fatal("Failed creating new request. %#v", err)
	}
	req.Header.Add("Content-Type", "application/json")
	dump, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		c.Fatal("Failed dumping request. %#v", err)
	}
	encoded := make([]byte, base64.StdEncoding.EncodedLen(len(dump)))
	base64.StdEncoding.Encode(encoded, dump)
	return encoded
}

func (s *ProxyTestSuite) setupWebsocketProxy() {
	// TODO Deduplicate. This method and the two below are close copies of the ones in logs_test.go.
	config.Parse()
	config.Config.HostUUID = "1"
	config.Config.ParsedPublicKey = wsp_utils.ParseTestPublicKey()
	s.privateKey = wsp_utils.ParseTestPrivateKey()

	conf := testutils.GetTestConfig(":4444")
	p := &proxy.Starter{
		BackendPaths:  []string{"/v1/connectbackend"},
		FrontendPaths: []string{"/v1/{dockersocket:dockersocket}/"},
		Config:        conf,
	}

	log.Infof("Starting websocket proxy. Listening on [%s].", conf.ListenAddr)

	go p.StartProxy()
	time.Sleep(time.Second)
	signedToken := wsp_utils.CreateBackendToken("1", s.privateKey)

	handlers := make(map[string]backend.Handler)
	handlers["/v1/dockersocket/"] = &Handler{}
	go backend.ConnectToProxy("ws://localhost:4444/v1/connectbackend?token="+signedToken, handlers)
	s.pullImage("ibuildthecloud/helloworld", "latest")
	time.Sleep(time.Second * time.Duration(2))
}

func (s *ProxyTestSuite) SetUpSuite(c *check.C) {
	cli, err := events.NewDockerClient()
	if err != nil {
		c.Fatalf("Could not connect to docker, err: [%v]", err)
	}
	s.client = cli
	s.setupWebsocketProxy()
}

func (s *ProxyTestSuite) pullImage(imageRepo, imageTag string) error {
	log.Infof("Pulling %v:%v image.", imageRepo, imageTag)
	_, err := s.client.ImagePull(context.Background(), imageRepo+":"+imageTag, types.ImagePullOptions{})
	return err
}
