package dockersocketproxy

import (
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/url"

	log "github.com/Sirupsen/logrus"
	"github.com/rancher/agent/service/hostapi/auth"
	"github.com/rancher/websocket-proxy/backend"
	"github.com/rancher/websocket-proxy/common"
)

type Handler struct {
}

func (s *Handler) Handle(key string, initialMessage string, incomingMessages <-chan string, response chan<- common.Message) {
	defer backend.SignalHandlerClosed(key, response)

	requestURL, err := url.Parse(initialMessage)
	if err != nil {
		log.WithFields(log.Fields{"error": err, "url": initialMessage}).Error("Couldn't parse url.")
		return
	}
	tokenString := requestURL.Query().Get("token")
	token, valid := auth.GetAndCheckToken(tokenString)
	if !valid {
		return
	}

	proto := "unix"
	address := "/var/run/docker.sock"

	if v, ok := token.Claims["proto"]; ok {
		proto = fmt.Sprint(v)
	}
	if v, ok := token.Claims["address"]; ok {
		address = fmt.Sprint(v)
	}

	conn, err := net.Dial(proto, address)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Errorf("Couldn't dial %s:%s", proto, address)
		return
	}
	log.Infof("Dialing %s:%s", proto, address)

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
				log.WithFields(log.Fields{"error": err}).Error("Error decoding message.")
				return
			}
			if _, err := conn.Write(data); err != nil {
				log.WithFields(log.Fields{"error": err}).Error("Error write message.")
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
				log.WithFields(log.Fields{"error": err}).Errorf("Error reading response.")
			}
			return
		}
	}
}
