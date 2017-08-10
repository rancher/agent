package proxy

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	"github.com/rancher/websocket-proxy/backend"
	"github.com/rancher/websocket-proxy/common"
	"github.com/rancher/websocket-proxy/testutils"
)

var privateKey interface{}

func TestMain(m *testing.M) {
	c := getTestConfig()
	privateKey = testutils.ParseTestPrivateKey()

	ps := &Starter{
		BackendPaths:       []string{"/v1/connectbackend"},
		FrontendPaths:      []string{"/v1/binaryecho", "/v1/echo", "/v1/oneanddone", "/v1/repeat", "/v1/sendafterclose"},
		StatsPaths:         []string{"/v1/hostStats/project"},
		CattleWSProxyPaths: []string{"/v1/subscribe", "/v1/wsproxyproto"},
		CattleProxyPaths:   []string{"/{cattle-proxy:.*}"},
		Config:             c,
	}
	go ps.StartProxy()

	handlers := make(map[string]backend.Handler)
	handlers["/v1/echo"] = &echoHandler{}
	handlers["/v1/binaryecho"] = &binaryEchoHandler{}
	handlers["/v1/oneanddone"] = &oneAndDoneHandler{}
	handlers["/v1/repeat"] = &repeatingHandler{}
	handlers["/v1/sendafterclose"] = &sendAfterCloseHandler{}
	handlers["/v1/hostStats/project"] = &statsHandler{1}
	signedToken := testutils.CreateBackendToken("1", privateKey)
	url := "ws://localhost:1111/v1/connectbackend?token=" + signedToken
	go backend.ConnectToProxy(url, handlers)

	signedToken = testutils.CreateBackendToken("2", privateKey)
	handlers2 := make(map[string]backend.Handler)
	handlers2["/v1/echo"] = &echoHandler{}
	handlers2["/v1/binaryecho"] = &binaryEchoHandler{}
	handlers2["/v1/oneanddone"] = &oneAndDoneHandler{}
	handlers2["/v1/repeat"] = &repeatingHandler{}
	handlers2["/v1/sendafterclose"] = &sendAfterCloseHandler{}
	handlers2["/v1/hostStats/project"] = &statsHandler{2}
	url = "ws://localhost:1111/v1/connectbackend?token=" + signedToken
	go backend.ConnectToProxy(url, handlers2)

	router := mux.NewRouter()
	router.HandleFunc("/v1/subscribe", getWsHandler())
	router.HandleFunc("/v1/proxyproto", proxyProtoHandler)
	router.HandleFunc("/v1/wsproxyproto", wsProxyProtoHandler)
	router.HandleFunc("/{foo:.*}", func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte("SUCCESS"))
	})
	go http.ListenAndServe("127.0.0.1:3333", router)

	time.Sleep(50 * time.Millisecond) // Give front and back a chance to initialize

	os.Exit(m.Run())
}

func proxyProtoHandler(rw http.ResponseWriter, req *http.Request) {
	xFor := req.Header.Get("X-Forwarded-For")
	xPort := req.Header.Get("X-Forwarded-Port")
	xProto := req.Header.Get("X-Forwarded-Proto")
	fmt.Fprintf(rw, "%s=%s,%s=%s,%s=%s\n", "xFor", xFor, "xPort", xPort, "xProto", xProto)
}

func wsProxyProtoHandler(rw http.ResponseWriter, req *http.Request) {
	xFor := req.Header.Get("X-Forwarded-For")
	xPort := req.Header.Get("X-Forwarded-Port")
	xProto := req.Header.Get("X-Forwarded-Proto")
	fmt.Fprintf(rw, "%s=%s,%s=%s,%s=%s\n", "xFor", xFor, "xPort", xPort, "xProto", xProto)
}

func getWsHandler() func(rw http.ResponseWriter, req *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		if !strings.EqualFold(req.Header.Get("Upgrade"), "websocket") {
			rw.Write([]byte("SUCCESS"))
			return
		}

		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}

		ws, err := upgrader.Upgrade(rw, req, nil)
		if err != nil {
			log.Fatal(err)
		}

		ws.WriteMessage(websocket.TextMessage, []byte("WSSUCCESS"))
		ws.Close()
	}
}

func TestEndToEnd(t *testing.T) {
	signedToken := testutils.CreateToken("1", privateKey)
	ws := getClientConnection("ws://localhost:1111/v1/echo?token="+signedToken, t)
	sendAndAssertReply(ws, strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10), t)
	time.Sleep(1 * time.Millisecond) // Ensure different timestamp
	sendAndAssertReply(ws, strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10), t)
}

func TestBinary(t *testing.T) {
	signedToken := testutils.CreateToken("1", privateKey)
	ws := getBinaryClientConnection("ws://localhost:1111/v1/binaryecho?token="+signedToken, t)
	sendBinaryAndAssertReply(ws, strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10), t)
	time.Sleep(1 * time.Millisecond) // Ensure different timestamp
	sendBinaryAndAssertReply(ws, strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10), t)
}

func TestAuthHeaderBearerToken(t *testing.T) {
	signedToken := testutils.CreateToken("1", privateKey)
	dialer := &websocket.Dialer{}
	headers := http.Header{}
	headers.Add("Authorization", "Bearer "+signedToken)
	ws, _, err := dialer.Dial("ws://localhost:1111/v1/echo", headers)
	if err != nil {
		t.Fatal(err)
	}
	sendAndAssertReply(ws, strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10), t)
	time.Sleep(1 * time.Millisecond) // Ensure different timestamp
	sendAndAssertReply(ws, strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10), t)
}

func TestBackendClosesConnection(t *testing.T) {
	signedToken := testutils.CreateToken("1", privateKey)
	ws := getClientConnection("ws://localhost:1111/v1/oneanddone?token="+signedToken, t)

	if err := ws.WriteMessage(1, []byte("a message")); err != nil {
		t.Fatal(err)
	}

	if _, _, err := ws.ReadMessage(); err != nil {
		t.Fatal(err)
	}

	if msgType, msgBytes, err := ws.ReadMessage(); err != io.EOF {
		t.Fatalf("Expected an EOF error to indicate connection was closed. [%v] [%s] [%v]", msgType, msgBytes, err)
	}
}

func TestFrontendClosesConnection(t *testing.T) {
	signedToken := testutils.CreateToken("1", privateKey)
	ws := getClientConnection("ws://localhost:1111/v1/oneanddone?token="+signedToken, t)
	if err := ws.WriteControl(websocket.CloseMessage, nil, time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}

	if _, _, err := ws.ReadMessage(); err == nil {
		t.Fatal("Expected error indicating websocket was closed.")
	}
}

func TestBackendSendAfterClose(t *testing.T) {
	signedToken := testutils.CreateToken("1", privateKey)
	ws := getClientConnection("ws://localhost:1111/v1/sendafterclose?token="+signedToken, t)
	go func() {
		for {
			_, _, err := ws.ReadMessage()
			if err != nil {
				return
			}
		}
	}()

	ws2 := getClientConnection("ws://localhost:1111/v1/echo?token="+signedToken, t)
	// If deadlock occurs, the read deadline will be hit and the sendAndAssertReply will fail the test
	ws2.SetReadDeadline(time.Now().Add(2 * time.Second))
	sendAndAssertReply(ws2, strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10), t)
}

func TestMultiHostStats(t *testing.T) {
	payload := map[string]interface{}{
		"project": []map[string]string{
			{
				"url":   "ws://localhost:1111/v1/hostStats/project",
				"token": testutils.CreateToken("1", privateKey),
			},
			{
				"url":   "ws://localhost:1111/v1/hostStats/project",
				"token": testutils.CreateToken("2", privateKey),
			},
		},
	}
	signedToken := testutils.CreateTokenWithPayload(payload, privateKey)
	ws := getClientConnection("ws://localhost:1111/v1/hostStats/project?token="+signedToken, t)
	one := false
	two := false
	for i := 0; i < 100; i++ {
		err := ws.WriteMessage(1, []byte("x"))
		if err != nil {
			t.Fatal("Error talking to host")
		}
		_, msgBytes, err := ws.ReadMessage()
		if err != nil {
			t.Fatal("Error reading response from various containers")
		}
		if string(msgBytes) == "1" {
			one = true
		}
		if string(msgBytes) == "2" {
			two = true
		}
		if one && two {
			return
		}
	}
	t.Fatal("Did not get container stats from two hosts")
}

func TestCattleProxy(t *testing.T) {
	resp, err := http.Get("http://localhost:1111/v1/foo1")
	assertProxyResponse(resp, err, t)

	req, err := http.NewRequest("PUT", "http://localhost:1111/v1///foo2", nil)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{}
	resp, err = client.Do(req)
	assertProxyResponse(resp, err, t)

	resp, err = http.Get("http://localhost:1111/v1/subscribe")
	assertProxyResponse(resp, err, t)
}

func TestCattleWsProxy(t *testing.T) {
	ws := getClientConnection("ws://localhost:1111/v1/subscribe", t)
	_, msg, err := ws.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if string(msg) != "WSSUCCESS" {
		t.Fatal("Unexpected message ", msg)
	}

}

func assertProxyResponse(resp *http.Response, err error, t *testing.T) {
	if err != nil || resp.StatusCode != 200 {
		t.Fatal("Bad response. ", resp, err)
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "SUCCESS" {
		t.Fatal("Unexpected body ", b)
	}
}

func getBinaryClientConnection(url string, t *testing.T) *websocket.Conn {
	headers := http.Header{}
	headers.Add("Sec-Websocket-Protocol", "binary")
	return getClientConnectionWithHeaders(url, t, headers)
}

func getClientConnection(url string, t *testing.T) *websocket.Conn {
	return getClientConnectionWithHeaders(url, t, http.Header{})
}

func getClientConnectionWithHeaders(url string, t *testing.T, headers http.Header) *websocket.Conn {
	dialer := &websocket.Dialer{}
	ws, _, err := dialer.Dial(url, headers)
	if err != nil {
		t.Fatal(err)
	}
	return ws
}

func sendAndAssertReply(ws *websocket.Conn, msg string, t *testing.T) {
	sendAndAssertReplyWithType(ws, msg, 1, t)
}

func sendBinaryAndAssertReply(ws *websocket.Conn, msg string, t *testing.T) {
	sendAndAssertReplyWithType(ws, msg, 2, t)
}

func sendAndAssertReplyWithType(ws *websocket.Conn, msg string, msgType int, t *testing.T) {
	message := []byte(msg)
	err := ws.WriteMessage(msgType, message)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Sent: %s\n", message)

	_, reply, err := ws.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Received: %s\n", reply)
	if msg+"-response" != string(reply) {
		t.Fatalf("Unexpected response: [%s]", reply)
	}
}

type sendAfterCloseHandler struct {
}

func (e *sendAfterCloseHandler) Handle(key string, initialMessage string, incomingMessages <-chan string, response chan<- common.Message) {
	backend.SignalHandlerClosed(key, response)
	m := "foo"
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

type statsHandler struct {
	i int
}

func (s *statsHandler) Handle(key string, initialMessage string, incomingMessages <-chan string, response chan<- common.Message) {
	defer backend.SignalHandlerClosed(key, response)
	data := fmt.Sprintf("%d", s.i)
	wrap := common.Message{
		Key:  key,
		Type: common.Body,
		Body: data,
	}
	response <- wrap
	return
}

type oneAndDoneHandler struct {
}

func (e *oneAndDoneHandler) Handle(key string, initialMessage string, incomingMessages <-chan string, response chan<- common.Message) {
	defer backend.SignalHandlerClosed(key, response)
	m := <-incomingMessages
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

type binaryEchoHandler struct {
}

func (e *binaryEchoHandler) Handle(key string, initialMessage string, incomingMessages <-chan string, response chan<- common.Message) {
	defer backend.SignalHandlerClosed(key, response)
	for {
		m, ok := <-incomingMessages
		if !ok {
			return
		}
		if m != "" {
			data, _ := base64.StdEncoding.DecodeString(m)
			resp := fmt.Sprintf("%s-response", data)
			respEncoded := base64.StdEncoding.EncodeToString([]byte(resp)[:])
			wrap := common.Message{
				Key:  key,
				Type: common.Body,
				Body: respEncoded,
			}
			response <- wrap
		}
	}
}

type echoHandler struct {
}

func (e *echoHandler) Handle(key string, initialMessage string, incomingMessages <-chan string, response chan<- common.Message) {
	defer backend.SignalHandlerClosed(key, response)
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

func getTestConfig() *Config {

	pubKey, err := ParsePublicKey("../testutils/public.pem")
	if err != nil {
		log.Fatal("Failed to parse key. ", err)
	}
	ports := map[int]bool{443: true}
	config := &Config{
		PublicKey:            pubKey,
		ListenAddr:           "127.0.0.1:1111",
		CattleAddr:           "127.0.0.1:3333",
		ProxyProtoHTTPSPorts: ports,
	}
	return config
}

func TestManyChattyConnections(t *testing.T) {
	// Spin up a hundred connections. The repeat handler will send a new message to each one
	// every 10 milliseconds. Stop after 5 seconds. This is just to prove that the proxy can handle a little load.
	for i := 1; i <= 100; i++ {
		signedToken := testutils.CreateToken("1", privateKey)
		msg := strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)
		ws := getClientConnection("ws://localhost:1111/v1/repeat?token="+signedToken+"&msg="+msg, t)
		go func(expectedPrefix string) {
			for {
				_, reply, err := ws.ReadMessage()
				if err != nil {
					t.Fatal(err)
				}
				if !strings.HasPrefix(string(reply), expectedPrefix) {
					t.Fatalf("Unexpected response: [%s]", reply)
				}
			}
		}(msg)
		time.Sleep(1 * time.Millisecond) // Ensure different timestamp
	}
	time.Sleep(5 * time.Second)
}

const wsProxyProtoTestRequest string = `GET /v1/wsproxyproto HTTP/1.1
Connection: Upgrade
Upgrade: websocket

`
const httpProxyProtoTestRequest string = "GET /v1/proxyproto HTTP/1.1\r\n\r\n"

func TestProxyProto(t *testing.T) {
	testProxyProto("127.0.0.1, 127.0.0.1", "2222", "http", httpProxyProtoTestRequest, t)
	testProxyProto("127.0.0.1, 127.0.0.1", "443", "https", httpProxyProtoTestRequest, t)
	testProxyProto("127.0.0.1", "443", "https", wsProxyProtoTestRequest, t)
}

func testProxyProto(forwardedFor string, forwardedPort string, forwardedProto string, body string, t *testing.T) {
	conn, err := net.Dial("tcp", "localhost:1111")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	clientIP, clientPort, err := net.SplitHostPort(conn.LocalAddr().String())
	if err != nil {
		t.Fatal(err)
	}
	fmt.Fprintf(conn, "PROXY TCP4 %s 1.1.1.1 %s %s\r\n%s", clientIP, clientPort, forwardedPort, body)
	expectedResult := fmt.Sprintf("xFor=%s,xPort=%s,xProto=%s", forwardedFor, forwardedPort, forwardedProto)
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "xFor") {
			if line != expectedResult {
				t.Fatalf("Unexpected result: [%s]", line)
			}
			break
		}
	}
}

func TestProxyProtocolHang(t *testing.T) {
	// Tests a bug where proxy would hang if a single empty connection connection was opened
	conn, err := net.Dial("tcp", "localhost:1111")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	fmt.Fprint(conn, "")

	gotResponse := make(chan bool)
	go func(resp chan bool) {
		http.Get("http://localhost:1111/v1/subscribe")
		resp <- true
	}(gotResponse)

	timer := time.NewTimer(time.Second)
	select {
	case <-gotResponse:
	case <-timer.C:
		t.FailNow()
	}

}

type repeatingHandler struct {
}

func (h *repeatingHandler) Handle(key string, initialMessage string, incomingMessages <-chan string, response chan<- common.Message) {
	u, err := url.Parse(initialMessage)
	if err != nil {
		log.Fatal(err)
	}
	msg := u.Query().Get("msg")
	idx := 0
	ticker := time.NewTicker(10 * time.Millisecond)
	for {
		select {
		case <-ticker.C:
			data := fmt.Sprintf("%s %d", msg, idx)
			wrap := common.Message{
				Key:  key,
				Type: common.Body,
				Body: data,
			}
			response <- wrap
		}
		idx++
	}
}
