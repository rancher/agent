package proxy

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/leodotcloud/log"
	"github.com/rancher/websocket-proxy/backend"
	"github.com/rancher/websocket-proxy/common"
)

var (
	httpClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
		Timeout: 60 * time.Second,
	}
)

type Handler struct {
}

func (s *Handler) Handle(key string, initialMessage string, incomingMessages <-chan string, response chan<- common.Message) {
	defer backend.SignalHandlerClosed(key, response)

	message, err := readMessage(incomingMessages)
	if err != nil {
		log.Error("Invalid content url=%v error=%v", initialMessage, err)
		return
	}

	log.Debugf("START %s: %#v url=%v", key, message, initialMessage)

	if message.Hijack {
		s.doHijack(message, key, incomingMessages, response)
	} else {
		s.doHTTP(message, key, incomingMessages, response)
	}
}

func (s *Handler) doHijack(message *common.HTTPMessage, key string, incomingMessages <-chan string, response chan<- common.Message) {
	req, err := http.NewRequest(message.Method, message.URL, nil)
	if err != nil {
		log.Errorf("Failed to create request error=%v", err)
		return
	}
	req.Host = message.Host
	req.Header = http.Header(message.Headers)

	if req.Header.Get("Connection") != "Upgrade" {
		req.Header.Set("Connection", "close")
	}

	content, err := setContentLength(req)
	if err != nil {
		return
	}

	u, err := url.Parse(message.URL)
	if err != nil {
		log.Errorf("Failed to parse URL %s error=%v", message.URL, err)
		return
	}

	var conn net.Conn
	if req.URL.Scheme == "https" || req.URL.Scheme == "wss" {
		conn, err = tls.Dial("tcp", u.Host, &tls.Config{
			InsecureSkipVerify: true,
		})
	} else {
		conn, err = net.Dial("tcp", u.Host)
	}
	if err != nil {
		log.Errorf("Failed to connect to %s error=%v", u.Host, err)
		return
	}
	defer conn.Close()

	reader := &HTTPReader{
		Buffered:   message.Body,
		Chan:       incomingMessages,
		EOF:        message.EOF,
		MessageKey: key,
	}

	writer := &HTTPWriter{
		MessageKey: key,
		Chan:       response,
	}

	if content > 0 {
		buf := make([]byte, content, content)
		if c, err := reader.Read(buf); err != nil || int64(c) != content {
			log.Errorf("Failed to read initial content for %s error=%v", u.Host, err)
		}
		req.Body = ioutil.NopCloser(bytes.NewReader(buf))
	}

	if err := req.Write(conn); err != nil {
		log.Errorf("Failed to write request to %s error=%v", u.Host, err)
		return
	}

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()
		if _, err := io.Copy(conn, reader); err != nil {
			log.Errorf("Failed to read request %s error=%v", u.Host, err)
		}
		reader.Close()

		for range incomingMessages {
			// waiting for channel to close
		}
		conn.Close()
	}()

	if _, err := io.Copy(writer, conn); err != nil {
		log.Errorf("Failed to write response for %s error=%v", u.Host, err)
	}
	writer.Close()

	wg.Wait()
}

func setContentLength(req *http.Request) (int64, error) {
	if lengthString := req.Header.Get("Content-Length"); lengthString != "" {
		length, err := strconv.Atoi(lengthString)
		if err != nil {
			log.Errorf("Failed to parse length %s error=%v", lengthString, err)
			return 0, err
		}
		req.ContentLength = int64(length)
	}
	return req.ContentLength, nil
}

func (s *Handler) doHTTP(message *common.HTTPMessage, key string, incomingMessages <-chan string, response chan<- common.Message) {
	req, err := http.NewRequest(message.Method, message.URL, &HTTPReader{
		Buffered:   message.Body,
		Chan:       incomingMessages,
		EOF:        message.EOF,
		MessageKey: key,
	})
	if err != nil {
		log.Errorf("Failed to create request error=%v", err)
		return
	}
	req.Host = message.Host
	req.Header = http.Header(message.Headers)

	if _, err := setContentLength(req); err != nil {
		return
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Errorf("Failed to make request error=%v", err)
		return
	}
	defer resp.Body.Close()

	httpResponseMessage := common.HTTPMessage{
		Code:    resp.StatusCode,
		Headers: map[string][]string(resp.Header),
	}

	httpWriter := &HTTPWriter{
		Message:    httpResponseMessage,
		MessageKey: key,
		Chan:       response,
	}
	defer httpWriter.Close()

	// Make sure we write the response codes if the response buffer is 0 bytes but blocking.
	// This happens with streaming logs a log
	if err := httpWriter.writeMessage(); err != nil {
		log.Errorf("Failed to write header error=%v", err)
		return
	}

	if _, err := io.Copy(httpWriter, resp.Body); err != nil {
		log.Errorf("Failed to write body error=%v", err)
		return
	}
}

func readMessage(incomingMessages <-chan string) (*common.HTTPMessage, error) {
	str := <-incomingMessages
	var message common.HTTPMessage
	if err := json.Unmarshal([]byte(str), &message); err != nil {
		return nil, err
	}
	return &message, nil
}
