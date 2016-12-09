package proxyprotocol

import (
	"net"
	"net/http"
	"strconv"
	"strings"
)

const (
	xForwardedProto string = "X-Forwarded-Proto"
	xForwardedPort  string = "X-Forwarded-Port"
	xForwardedFor   string = "X-Forwarded-For"
	sep             string = ", "
)

func AddHeaders(req *http.Request, httpsPorts map[int]bool) {
	proxyProtoInfo := getInfo(req.RemoteAddr)
	if proxyProtoInfo != nil {
		if h := req.Header.Get(xForwardedProto); h == "" {
			var proto string
			if _, ok := httpsPorts[proxyProtoInfo.ProxyAddr.Port]; ok {
				proto = "https"
			} else {
				proto = "http"
			}
			req.Header.Set(xForwardedProto, proto)
		}

		if h := req.Header.Get(xForwardedPort); h == "" {
			req.Header.Set(xForwardedPort, strconv.Itoa(proxyProtoInfo.ProxyAddr.Port))
		}

		ip := proxyProtoInfo.ClientAddr.IP.String()
		if forwardedFors, ok := req.Header[http.CanonicalHeaderKey(xForwardedFor)]; ok {
			ip = strings.Join(forwardedFors, sep) + sep + ip
		}
		req.Header.Set(xForwardedFor, ip)

	} else if req.TLS != nil {
		if h := req.Header.Get(xForwardedProto); h == "" {
			req.Header.Set(xForwardedProto, "https")
		}
	}
}

func AddForwardedFor(req *http.Request) {
	ip := strings.Split(req.RemoteAddr, ":")[0]
	if forwardedFors, ok := req.Header[http.CanonicalHeaderKey(xForwardedFor)]; ok {
		ip = strings.Join(forwardedFors, sep) + sep + ip
	}
	req.Header.Set(xForwardedFor, ip)
}

func StateCleanup(conn net.Conn, connState http.ConnState) {
	if connState == http.StateClosed {
		deleteInfo(conn.RemoteAddr().String())
	}
}
