package proxy

import (
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/websocket"
	"github.com/pborman/uuid"

	"github.com/rancher/websocket-proxy/common"
)

type multiplexer struct {
	backendSessionID  string
	backendKey        string
	messagesToBackend chan string
	frontendChans     map[string]chan<- common.Message
	proxyManager      proxyManager
	frontendMu        *sync.RWMutex
}

func (m *multiplexer) initializeClient() (string, <-chan common.Message) {
	msgKey := uuid.New()
	frontendChan := make(chan common.Message)
	m.frontendMu.Lock()
	defer m.frontendMu.Unlock()
	m.frontendChans[msgKey] = frontendChan
	return msgKey, frontendChan
}

func (m *multiplexer) connect(msgKey, url string) {
	m.messagesToBackend <- common.FormatMessage(msgKey, common.Connect, url)
}

func (m *multiplexer) send(msgKey, msg string) {
	m.messagesToBackend <- common.FormatMessage(msgKey, common.Body, msg)
}

func (m *multiplexer) sendClose(msgKey string) {
	m.messagesToBackend <- common.FormatMessage(msgKey, common.Close, "")
}

func (m *multiplexer) closeConnection(msgKey string, notifyBackend bool) {
	if notifyBackend {
		m.sendClose(msgKey)
	}

	m.frontendMu.Lock()
	defer m.frontendMu.Unlock()
	if frontendChan, ok := m.frontendChans[msgKey]; ok {
		close(frontendChan)
		delete(m.frontendChans, msgKey)
	}
}

func (m *multiplexer) routeMessages(ws *websocket.Conn) {
	stopSignal := make(chan bool, 1)

	// Read messages from backend
	go func(stop chan<- bool) {
		for {
			msgType, msg, err := ws.ReadMessage()
			if err != nil {
				log.Infof("Shutting down backend %v. Connection closed because: %v.", m.backendKey, err)
				m.shutdown(stop)
				return
			}

			if msgType != websocket.TextMessage {
				continue
			}
			message := common.ParseMessage(string(msg))

			m.frontendMu.RLock()
			frontendChan, ok := m.frontendChans[message.Key]
			timedOut := false
			if ok {
				select {
				case frontendChan <- message:
				case <-time.After(time.Second * 10):
					timedOut = true
				}
			}
			m.frontendMu.RUnlock()

			if timedOut {
				log.Warnf("Timed out sending message with key %v to frontend channel.", message.Key)
				m.proxyManager.closeConnection(m.backendKey, message.Key)
			}

			if !ok && message.Type != common.Close {
				log.Infof("Couldn't find frontend channel for key %v. Closing frontend connection.", m.backendKey)
				m.proxyManager.closeConnection(m.backendKey, message.Key)
			}
		}
	}(stopSignal)

	// Write messages to backend
	go func(stop <-chan bool) {
		ticker := time.NewTicker(time.Second * 5)
		defer ticker.Stop()
		for {
			select {
			case message, ok := <-m.messagesToBackend:
				if !ok {
					return
				}
				ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
				err := ws.WriteMessage(websocket.TextMessage, []byte(message))
				if err != nil {
					log.Errorf("Error writing message to backend %v - %v. Error: %v", m.backendKey, m.backendSessionID, err)
					ws.Close()
				}
			case <-ticker.C:
				ws.WriteControl(websocket.PingMessage, []byte(""), time.Now().Add(time.Second))

			case <-stop:
				return
			}
		}
	}(stopSignal)
}

func (m *multiplexer) shutdown(stop chan<- bool) {
	m.proxyManager.removeBackend(m.backendKey, m.backendSessionID)
	stop <- true
	for key := range m.frontendChans {
		m.closeConnection(key, false)
	}
}
