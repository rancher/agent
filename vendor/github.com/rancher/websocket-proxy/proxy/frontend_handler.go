package proxy

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gorilla/websocket"

	"github.com/rancher/websocket-proxy/common"
)

const wsProto string = "Sec-Websocket-Protocol"
const wsProtoBinary string = "binary"

type FrontendHandler struct {
	backend         backendProxy
	parsedPublicKey interface{}
}

func (h *FrontendHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	_, hostKey, authErr := h.auth(req)
	if authErr != nil {
		log.Infof("Frontend auth failed: %v", authErr)
		http.Error(rw, "Failed authentication", 401)
		return
	}

	binary := strings.EqualFold(req.Header.Get(wsProto), wsProtoBinary)
	respHeaders := make(http.Header)
	if binary {
		respHeaders.Add(wsProto, wsProtoBinary)
	}
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	ws, err := upgrader.Upgrade(rw, req, respHeaders)
	if err != nil {
		log.Errorf("Error during upgrade: [%v]", err)
		http.Error(rw, "Failed to upgrade connection.", 500)
		return
	}
	defer closeConnection(ws)

	msgKey, respChannel, err := h.backend.initializeClient(hostKey)
	if err != nil {
		log.Errorf("Error during initialization: [%v]", err)
		closeConnection(ws)
		return
	}
	defer h.backend.closeConnection(hostKey, msgKey)

	// Send response messages to client
	go func() {
		defer closeConnection(ws)
		for {
			message, ok := <-respChannel
			if !ok {
				return
			}
			switch message.Type {
			case common.Body:
				var data []byte
				var e error
				msgType := 1
				if binary {
					msgType = 2
					data, e = base64.StdEncoding.DecodeString(message.Body)
					if e != nil {
						log.Errorf("Error decoding message: %v", e)
						closeConnection(ws)
						continue
					}
				} else {
					data = []byte(message.Body)
				}

				ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if e := ws.WriteMessage(msgType, data); e != nil {
					closeConnection(ws)
				}
			case common.Close:
				closeConnection(ws)
			}
		}
	}()

	url := req.URL.String()
	if err = h.backend.connect(hostKey, msgKey, url); err != nil {
		return
	}

	// Send request messages to backend
	for {
		msgType, msg, err := ws.ReadMessage()
		if err != nil {
			return
		}
		var data string
		if binary {
			data = base64.StdEncoding.EncodeToString(msg)
		} else {
			data = string(msg)
		}
		if msgType == websocket.BinaryMessage || msgType == websocket.TextMessage {
			if err = h.backend.send(hostKey, msgKey, data); err != nil {
				return
			}
		}
	}
}

func (h *FrontendHandler) auth(req *http.Request) (*jwt.Token, string, error) {
	token, tokenParam, err := parseToken(req, h.parsedPublicKey)
	if err != nil {
		if tokenParam == "" {
			return nil, "", noTokenError{}
		}
		return nil, "", fmt.Errorf("Error parsing token: %v. Token parameter: %v", err, tokenParam)
	}

	if !token.Valid {
		return nil, "", fmt.Errorf("Token not valid. Token parameter: %v.", tokenParam)
	}

	hostUUID, found := token.Claims["hostUuid"]
	if found {
		if hostKey, ok := hostUUID.(string); ok && h.backend.hasBackend(hostKey) {
			return token, hostKey, nil
		}
	}

	return nil, "", fmt.Errorf("Invalid backend host requested: %v.", hostUUID)
}

func closeConnection(ws *websocket.Conn) {
	ws.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
	ws.Close()
}

func parseToken(req *http.Request, parsedPublicKey interface{}) (*jwt.Token, string, error) {
	tokenString := ""
	if authHeader := req.Header.Get("Authorization"); authHeader != "" {
		if len(authHeader) > 6 && strings.EqualFold("bearer", authHeader[0:6]) {
			tokenString = strings.Trim(authHeader[7:], " ")
		}
	}

	if tokenString == "" {
		tokenString = req.URL.Query().Get("token")
	}

	if tokenString == "" {
		return nil, "", fmt.Errorf("No JWT provided")
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return parsedPublicKey, nil
	})
	return token, tokenString, err
}

type noTokenError struct {
}

func (e noTokenError) Error() string {
	return "Request did not have a token parameter."
}

func IsNoTokenError(err error) bool {
	_, ok := err.(noTokenError)
	return ok
}
