package proxy

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/websocket-proxy/common"
)

type BackendHTTPReader struct {
	backend         backendProxy
	hostKey, msgKey string
	messages        <-chan common.Message
	data            chan string
	buffer          []byte
	rw              http.ResponseWriter
}

func NewBackendHTTPReader(rw http.ResponseWriter, hostKey, msgKey string, backend backendProxy, messages <-chan common.Message) *BackendHTTPReader {
	b := &BackendHTTPReader{
		hostKey:  hostKey,
		msgKey:   msgKey,
		messages: messages,
		data:     make(chan string),
		backend:  backend,
		rw:       rw,
	}
	go b.start()
	return b
}

func (b *BackendHTTPReader) start() {
	logrus.Debugf("BACKEND READER OPEN %s", b.msgKey)
	closed := false
	defer close(b.data)

	for message := range b.messages {
		if closed {
			// Just consume messages at this point
			continue
		}

		switch message.Type {
		case common.Body:
			b.data <- message.Body
		case common.Close:
			logrus.Debugf("BACKEND CLOSE RECIEVED %s", b.msgKey)
			closed = true
			b.backend.closeConnection(b.hostKey, b.msgKey)
		}
	}
	logrus.Debugf("BACKEND READER CLOSE %s", b.msgKey)
}

func (b *BackendHTTPReader) Close() error {
	logrus.Debugf("BACKEND CLOSE REQUESTED %s", b.msgKey)
	go b.backend.closeConnection(b.hostKey, b.msgKey)
	for range b.data {
		// Make sure the start() go routine is closed
	}
	return nil
}

func (b *BackendHTTPReader) Read(out []byte) (int, error) {
	if len(b.buffer) == 0 {
		message, ok := <-b.data
		if !ok {
			logrus.Debugf("BACKEND READ CHANNEL EOF: %s %s", b.hostKey, b.msgKey)
			return 0, io.EOF
		}

		var response common.HTTPMessage
		if err := json.Unmarshal([]byte(message), &response); err != nil {
			logrus.Errorf("%s %s: %v", b.hostKey, b.msgKey, err)
			return 0, err
		}

		if response.EOF {
			logrus.Debugf("BACKEND READ RESPONSE EOF: %s %s", b.hostKey, b.msgKey)
			return 0, io.EOF
		}

		b.buffer = []byte(response.Body)

		for k, v := range response.Headers {
			logrus.Debugf("BACKEND READ HEADER %s %s %s %v", b.hostKey, b.msgKey, k, v)
			b.rw.Header()[k] = v
		}

		if response.Code > 0 && b.rw != nil {
			logrus.Debugf("BACKEND READ STATUS CODE: %s %s %d", b.hostKey, b.msgKey, response.Code)
			b.rw.WriteHeader(response.Code)
			flush(b.rw)
		}
	}

	c := copy(out, b.buffer)
	b.buffer = b.buffer[c:]
	logrus.Debugf("BACKEND READ %s: %s buffer: %s", b.msgKey, out[:c], b.buffer)
	return c, nil
}
