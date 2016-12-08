package backend

import (
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/websocket"
)

func newPongHandler(ws *websocket.Conn) *pongHandler {
	ph := &pongHandler{
		mu:       &sync.Mutex{},
		lastPing: time.Now(),
		ws:       ws,
	}

	go ph.startTimer(5000, 10000)

	return ph
}

type pongHandler struct {
	mu       *sync.Mutex
	lastPing time.Time
	ws       *websocket.Conn
}

func (h *pongHandler) startTimer(checkInterval, maxWait int) {
	ticker := time.NewTicker(time.Millisecond * time.Duration(checkInterval))
	defer ticker.Stop()
	for range ticker.C {
		h.mu.Lock()
		t := h.lastPing
		timeoutAt := t.Add(time.Millisecond * time.Duration(maxWait))
		h.mu.Unlock()
		if time.Now().After(timeoutAt) {
			logrus.Warnf("Hit websocket pong timeout. Last websocket ping received at %v. Closing connection.", t)
			h.ws.Close()
		}
	}
}

func (h *pongHandler) handle(appData string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.lastPing = time.Now()
	return nil
}
