package proxy

import (
	"encoding/json"
	"io"

	"github.com/leodotcloud/log"
	"github.com/rancher/websocket-proxy/common"
)

type HTTPWriter struct {
	headerWritten bool
	Message       common.HTTPMessage
	MessageKey    string
	Chan          chan<- common.Message
}

func (h *HTTPWriter) Write(bytes []byte) (n int, err error) {
	h.Message.Body = bytes
	if err := h.writeMessage(); err != nil {
		return 0, err
	}
	return len(bytes), nil
}

func (h *HTTPWriter) writeMessage() error {
	bytes, err := json.Marshal(&h.Message)
	if err != nil {
		return err
	}
	m := common.Message{
		Key:  h.MessageKey,
		Type: common.Body,
		Body: string(bytes),
	}
	log.Debugf("HTTP WRITER %s: %#v", h.MessageKey, m)
	h.Chan <- m
	h.Message = common.HTTPMessage{}
	return nil
}

func (h *HTTPWriter) Close() error {
	h.Message.EOF = true
	return h.writeMessage()
}

type HTTPReader struct {
	Buffered   []byte
	Chan       <-chan string
	EOF        bool
	MessageKey string
}

func (h *HTTPReader) Close() error {
	log.Debugf("HTTP READER CLOSE %s", h.MessageKey)
	return nil
}

func (h *HTTPReader) Read(bytes []byte) (int, error) {
	if len(h.Buffered) == 0 && !h.EOF {
		if err := h.read(); err != nil {
			return 0, err
		}
	}

	count := copy(bytes, h.Buffered)
	h.Buffered = h.Buffered[count:]

	if h.EOF {
		log.Debugf("HTTP READER RETURN EOF %s", h.MessageKey)
		return count, io.EOF
	}
	log.Debugf("HTTP READER RETURN COUNT %s %d %d: %s", h.MessageKey, count, len(h.Buffered), bytes[:count])
	return count, nil
}

func (h *HTTPReader) read() error {
	str, ok := <-h.Chan
	if !ok {
		log.Debugf("HTTP READER CHANNEL EOF %s", h.MessageKey)
		return io.EOF
	}

	var message common.HTTPMessage
	if err := json.Unmarshal([]byte(str), &message); err != nil {
		return err
	}

	log.Debugf("HTTP READER MESSAGE %s %s", h.MessageKey, message.Body)

	h.Buffered = message.Body
	h.EOF = message.EOF
	return nil
}
