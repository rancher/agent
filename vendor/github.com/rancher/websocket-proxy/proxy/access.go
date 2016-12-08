// Package proxy is inspired by https://gist.github.com/cespare/3985516
package proxy

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
)

type accessLog struct {
	ip, method, uri, protocol, host string
	elapsedTime                     time.Duration
}

func logAccess(w http.ResponseWriter, req *http.Request, duration time.Duration) {
	clientIP := req.RemoteAddr

	if colon := strings.LastIndex(clientIP, ":"); colon != -1 {
		clientIP = clientIP[:colon]
	}

	record := &accessLog{
		ip:          clientIP,
		method:      req.Method,
		uri:         req.RequestURI,
		protocol:    req.Proto,
		host:        req.Host,
		elapsedTime: duration,
	}

	writeAccessLog(record)
}

func writeAccessLog(record *accessLog) {
	logRecord := "" + record.ip + " " + record.protocol + " " + record.method + ": " + record.uri + ", host: " + record.host + " (load time: " + strconv.FormatFloat(record.elapsedTime.Seconds(), 'f', 5, 64) + " seconds)"
	log.Info(logRecord)
}
