package proxy

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/rancher/websocket-proxy/common"
	"sync"
)

type BackendHTTPWriter struct {
	hostKey, msgKey string
	backend         backendProxy
	mu              sync.Mutex
	closed          bool
}

func (b *BackendHTTPWriter) Close() error {
	b.mu.Lock()
	if b.closed {
		return nil
	}
	b.closed = true
	b.mu.Unlock()

	logrus.Debugf("BACKEND WRITE EOF %s", b.msgKey)
	return b.writeMessage(&common.HTTPMessage{
		EOF: true,
	})
}

func (b *BackendHTTPWriter) WriteRequest(req *http.Request, hijack bool, address, scheme string) error {
	vars := mux.Vars(req)

	headers := http.Header{}
	for k, v := range req.Header {
		headers[k] = v
	}

	url := *req.URL

	if req.TLS == nil {
		url.Scheme = "http"
	} else {
		url.Scheme = "https"
	}
	url.Host = req.Host
	headers.Set("X-API-request-url", url.String())

	url.Host = address
	url.Path = vars["path"]
	if !strings.HasPrefix(url.Path, "/") {
		url.Path = "/" + url.Path
	}

	if scheme == "" {
		url.Scheme = "http"
	} else {
		url.Scheme = scheme
	}

	return b.writeMessage(&common.HTTPMessage{
		Hijack:  hijack,
		Host:    req.Host,
		Method:  req.Method,
		URL:     url.String(),
		Headers: headers,
	})
}

func (b *BackendHTTPWriter) writeMessage(message *common.HTTPMessage) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	logrus.Debugf("BACKEND WRITE %s,%s: %s", b.hostKey, b.msgKey, data)
	return b.backend.send(b.hostKey, b.msgKey, string(data))
}

func (b *BackendHTTPWriter) Write(buffer []byte) (int, error) {
	return len(buffer), b.writeMessage(&common.HTTPMessage{
		Body: buffer,
	})
}
