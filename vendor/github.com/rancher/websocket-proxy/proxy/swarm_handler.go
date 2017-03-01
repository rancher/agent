package proxy

import "net/http"

type SwarmHandler struct {
	FrontendHandler http.Handler
	DefaultHandler  http.Handler
}

func (s *SwarmHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if req.TLS != nil && len(req.TLS.PeerCertificates) > 0 {
		s.FrontendHandler.ServeHTTP(rw, req)
	} else {
		s.DefaultHandler.ServeHTTP(rw, req)
	}
}
