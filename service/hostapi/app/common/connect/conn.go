package connect

import (
	"github.com/gorilla/websocket"
	"net/http"
)

type connection struct {
	webConn *websocket.Conn
	rw      http.ResponseWriter
}

func (conn *connection) Write(data []byte) (int, error) {
	if conn.webConn == nil {
		count, err := conn.rw.Write(data)
		if err != nil {
			return count, err
		}

		if flusher, ok := conn.rw.(http.Flusher); ok {
			flusher.Flush()
		}
	} else {
		err := conn.webConn.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			return 0, err
		}
	}

	return len(data), nil
}

func (conn *connection) IsContinuous() bool {
	return conn.webConn != nil
}
