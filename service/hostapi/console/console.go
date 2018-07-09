package console

import (
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/url"

	"github.com/rancher/agent/service/hostapi/auth"
	"github.com/rancher/log"
	"github.com/rancher/websocket-proxy/backend"
	"github.com/rancher/websocket-proxy/common"
)

const socketLocFmt string = "/var/lib/rancher/vm/%v/vnc"

type Handler struct {
}

func (s *Handler) Handle(key string, initialMessage string, incomingMessages <-chan string, response chan<- common.Message) {
	defer backend.SignalHandlerClosed(key, response)
	requestURL, err := url.Parse(initialMessage)
	if err != nil {
		log.Errorf("Couldn't parse url. url=%v error=%v", initialMessage, err)
		return
	}
	tokenString := requestURL.Query().Get("token")
	token, valid := auth.GetAndCheckToken(tokenString)
	if !valid {
		return
	}

	console := token.Claims["console"].(map[string]interface{})
	container := console["container"].(string)

	socketLoc := fmt.Sprintf(socketLocFmt, container)
	conn, err := net.Dial("unix", socketLoc)
	if err != nil {
		log.Errorf("Couldn't dial VM socket [%v]. error=%v", socketLoc, err)
		return
	}

	closed := false
	go func() {
		defer func() {
			closed = true
			conn.Close()
		}()

		for {
			msg, ok := <-incomingMessages
			if !ok {
				return
			}
			data, err := base64.StdEncoding.DecodeString(msg)

			if err != nil {
				log.Errorf("Error decoding message. error=%v", err)
				return
			}
			if _, err := conn.Write(data); err != nil {
				log.Errorf("Error write message. error=%v", err)
				return
			}
		}
	}()

	for {
		buff := make([]byte, 1024)
		n, err := conn.Read(buff)
		if n > 0 && err == nil {
			text := base64.StdEncoding.EncodeToString(buff[:n])
			message := common.Message{
				Key:  key,
				Type: common.Body,
				Body: text,
			}
			response <- message
		}
		if err != nil {
			if err != io.EOF && !closed {
				log.Errorf("Error reading response. error=%v", err)
			}
			return
		}
	}
}
