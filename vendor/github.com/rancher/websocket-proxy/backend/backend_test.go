package backend

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/websocket"

	"github.com/rancher/websocket-proxy/common"
	"github.com/rancher/websocket-proxy/proxy"
	"github.com/rancher/websocket-proxy/testutils"
)

var privateKey interface{}

func TestMain(m *testing.M) {
	c := getTestConfig()
	privateKey = testutils.ParseTestPrivateKey()

	ps := &proxy.Starter{
		BackendPaths:  []string{"/v1/connectbackend"},
		FrontendPaths: []string{"/v1/echo"},
		Config:        c,
	}
	go ps.StartProxy()

	os.Exit(m.Run())
}

func TestBackendGoesAway(t *testing.T) {
	dialer := &websocket.Dialer{}
	headers := http.Header{}
	signedToken := testutils.CreateBackendToken("1", privateKey)
	url := "ws://localhost:2223/v1/connectbackend?token=" + signedToken
	backendWs, _, err := dialer.Dial(url, headers)
	if err != nil {
		t.Fatal("Failed to connect to proxy.", err)
	}

	handlers := make(map[string]Handler)
	handlers["/v1/echo"] = &echoHandler{}
	go connectToProxyWS(backendWs, handlers)

	signedToken = testutils.CreateToken("1", privateKey)
	url = "ws://localhost:2223/v1/echo?token=" + signedToken
	ws := getClientConnection(url, t)

	if err := ws.WriteMessage(1, []byte("a message")); err != nil {
		t.Fatal(err)
	}

	ws.ReadMessage() // Read initial echo message
	backendWs.Close()

	if _, msg, err := ws.ReadMessage(); err != io.EOF {
		t.Fatalf("Expected error indicating websocket was closed. Received: %s", msg)
	}

	dialer = &websocket.Dialer{}
	ws, _, err = dialer.Dial(url, http.Header{})
	if ws != nil || err != websocket.ErrBadHandshake {
		t.Fatal("Should not have been able to connect.")
	}
}

func TestBackendReplaced(t *testing.T) {
	// This tests that if a backend connection A is replaced by backend connection B and then A is closed, the
	// multiplexer for B is not lost or removed.
	dialer := &websocket.Dialer{}
	url := "ws://localhost:2223/v1/connectbackend?token=" + testutils.CreateBackendToken("1", privateKey)
	backendWs, _, err := dialer.Dial(url, http.Header{})
	if err != nil {
		t.Fatal("Failed to connect to proxy.", err)
	}

	backendWs2, _, err := dialer.Dial(url, http.Header{})
	if err != nil {
		t.Fatal("Failed to connect to proxy second time.", err)
	}
	defer backendWs2.Close()

	go connectToProxyWS(backendWs2, map[string]Handler{"/v1/echo": &echoHandler{}})

	backendWs.Close()

	ws := getClientConnection("ws://localhost:2223/v1/echo?token="+testutils.CreateToken("1", privateKey), t)
	if err := ws.WriteMessage(1, []byte("a message")); err != nil {
		t.Fatal(err)
	}

	_, msg, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("Unexpected error: %s", msg)
	}

	if string(msg) != "a message-response" {
		t.Fatalf("Unexpected message: %s", msg)
	}
}

// Simple unit test for asserting the GetHandler algorithm
func TestGetHandler(t *testing.T) {
	handlers := map[string]Handler{}
	logKey := "/v1/logs/"
	statKey := "/v1/stats/"
	handlers[logKey] = &mockHandler{hType: logKey}
	handlers[statKey] = &mockHandler{hType: statKey}

	if !assertHandler("/v1/logs", logKey, handlers, t) {
		t.Fatal("Bad handler")
	}
	if !assertHandler("/v1/logs/", logKey, handlers, t) {
		t.Fatal("Bad handler")
	}
	if !assertHandler("/v1/stats/", statKey, handlers, t) {
		t.Fatal("Bad handler")
	}
	if !assertHandler("/v1/stats", statKey, handlers, t) {
		t.Fatal("Bad handler")
	}
	if !assertHandler("/v1/stats/1234", statKey, handlers, t) {
		t.Fatal("Bad handler")
	}
	if !assertHandler("/v1/stats/1234/", statKey, handlers, t) {
		t.Fatal("Bad handler")
	}
	if assertHandler("/v1/foo", statKey, handlers, t) {
		t.Fatal("Bad handler")
	}
}

func assertHandler(path string, expectedType string, handlers map[string]Handler, t *testing.T) bool {
	if h, ok := getHandler(path, handlers); ok {
		if mh, yes := h.(*mockHandler); yes && mh.hType == expectedType {
			return true
		}
	}
	return false
}

type mockHandler struct {
	hType string
}

func (h *mockHandler) Handle(messageKey string, initialMessage string, incomingMessages <-chan string, response chan<- common.Message) {

}

func getClientConnection(url string, t *testing.T) *websocket.Conn {
	dialer := &websocket.Dialer{}
	ws, _, err := dialer.Dial(url, http.Header{})
	if err != nil {
		t.Fatal(err)
	}
	return ws
}

type echoHandler struct {
}

func (e *echoHandler) Handle(key string, initialMessage string, incomingMessages <-chan string, response chan<- common.Message) {
	defer SignalHandlerClosed(key, response)
	for {
		m, ok := <-incomingMessages
		if !ok {
			return
		}
		if m != "" {
			data := fmt.Sprintf("%s-response", m)
			wrap := common.Message{
				Key:  key,
				Type: common.Body,
				Body: data,
			}
			response <- wrap
		}
	}
}

func getTestConfig() *proxy.Config {
	config := &proxy.Config{
		ListenAddr: "127.0.0.1:2223",
		CattleAddr: "127.0.0.1:8081",
	}

	pubKey, err := proxy.ParsePublicKey("../testutils/public.pem")
	if err != nil {
		log.Fatal("Failed to parse key. ", err)
	}
	config.PublicKey = pubKey
	return config
}
