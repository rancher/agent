package proxy

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/docker/docker/pkg/tlsconfig"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"

	"github.com/rancher/websocket-proxy/k8s"
	"github.com/rancher/websocket-proxy/proxy/proxyprotocol"
	proxyTls "github.com/rancher/websocket-proxy/proxy/tls"
)

var slashRegex = regexp.MustCompile("[/]{2,}")

type Starter struct {
	BackendPaths       []string
	FrontendPaths      []string
	FrontendHTTPPaths  []string
	StatsPaths         []string
	CattleProxyPaths   []string
	CattleWSProxyPaths []string
	Config             *Config
}

func (s *Starter) StartProxy() error {
	backendMultiplexers := make(map[string]*multiplexer)
	bpm := &backendProxyManager{
		multiplexers: backendMultiplexers,
		mu:           &sync.RWMutex{},
	}

	frontendHandler := &FrontendHandler{
		backend:         bpm,
		parsedPublicKey: s.Config.PublicKey,
	}

	statsHandler := &StatsHandler{
		backend:         bpm,
		parsedPublicKey: s.Config.PublicKey,
	}

	backendHandler := &BackendHandler{
		proxyManager:    bpm,
		parsedPublicKey: s.Config.PublicKey,
	}

	frontendHTTPHandler := &FrontendHTTPHandler{
		FrontendHandler: FrontendHandler{
			backend:         bpm,
			parsedPublicKey: s.Config.PublicKey,
		},
		HTTPSPorts:  s.Config.ProxyProtoHTTPSPorts,
		TokenLookup: NewTokenLookup(s.Config.CattleAddr),
	}

	cattleProxy, cattleWsProxy := newCattleProxies(s.Config)

	router := mux.NewRouter()

	router.HandleFunc("/version", k8s.Version)
	router.HandleFunc("/swaggerapi/api/v1", k8s.Swagger)

	for _, p := range s.BackendPaths {
		router.Handle(p, backendHandler).Methods("GET")
	}
	for _, p := range s.FrontendPaths {
		router.Handle(p, frontendHandler).Methods("GET")
	}
	for _, p := range s.FrontendHTTPPaths {
		router.Handle(p, frontendHTTPHandler).Methods("GET", "POST", "PUT", "DELETE", "PATCH")
	}
	for _, p := range s.StatsPaths {
		router.Handle(p, statsHandler).Methods("GET")
	}

	if s.Config.CattleAddr != "" {
		for _, p := range s.CattleWSProxyPaths {
			router.Handle(p, cattleWsProxy)
		}

		for _, p := range s.CattleProxyPaths {
			router.Handle(p, cattleProxy)
		}
	}

	if s.Config.ParentPid != 0 {
		go func() {
			for {
				process, err := os.FindProcess(s.Config.ParentPid)
				if err != nil {
					log.Fatalf("Failed to find process: %s\n", err)
				} else {
					err := process.Signal(syscall.Signal(0))
					if err != nil {
						log.Fatal("Parent process went away. Shutting down.")
					}
				}
				time.Sleep(time.Millisecond * 250)
			}
		}()
	}

	pcRouter := &pathCleaner{
		router: router,
	}

	swarmHandler := &SwarmHandler{
		FrontendHandler: frontendHTTPHandler,
		DefaultHandler:  pcRouter,
	}

	server := &http.Server{
		Handler:   swarmHandler,
		Addr:      s.Config.ListenAddr,
		ConnState: proxyprotocol.StateCleanup,
	}

	listener, err := net.Listen("tcp", s.Config.ListenAddr)
	if err != nil {
		log.Fatalf("Couldn't create listener: %s\n", err)
	}

	listener = &proxyprotocol.Listener{listener}

	if s.Config.TLSListenAddr != "" {
		tlsConfig, err := s.setupTLS()
		if err != nil {
			return err
		}

		if s.Config.TLSListenAddr == s.Config.ListenAddr {
			listener = &proxyTls.SplitListener{
				Listener: listener,
				Config:   tlsConfig,
			}
		} else {
			tlsListener, err := net.Listen("tcp", s.Config.TLSListenAddr)
			if err != nil {
				return err
			}
			tlsListener = &proxyprotocol.Listener{tlsListener}
			go func() {
				defer listener.Close()
				log.Error(server.Serve(tls.NewListener(tlsListener, tlsConfig)))
			}()
		}
	}

	err = server.Serve(listener)
	return err
}

func (s *Starter) setupTLS() (*tls.Config, error) {
	if s.Config.CattleAccessKey == "" {
		return nil, fmt.Errorf("No access key supplied to download cert")
	}

	certs, err := s.Config.GetCerts()
	if err != nil {
		return nil, err
	}

	tlsCert, err := tls.X509KeyPair(certs.Cert, certs.Key)
	if err != nil {
		return nil, err
	}

	clientCas := x509.NewCertPool()
	if !clientCas.AppendCertsFromPEM(certs.CA) {
		return nil, err
	}

	tlsConfig := tlsconfig.ServerDefault()
	tlsConfig.ClientAuth = tls.VerifyClientCertIfGiven
	tlsConfig.ClientCAs = clientCas
	tlsConfig.Certificates = []tls.Certificate{tlsCert}

	return tlsConfig, nil
}

type pathCleaner struct {
	router *mux.Router
}

func (p *pathCleaner) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if cleanedPath := p.cleanPath(req.URL.Path); cleanedPath != req.URL.Path {
		req.URL.Path = cleanedPath
		req.URL.Scheme = "http"
	}
	p.router.ServeHTTP(rw, req)
}

func (p *pathCleaner) cleanPath(path string) string {
	return slashRegex.ReplaceAllString(path, "/")
}

func newCattleProxies(config *Config) (*proxyProtocolConverter, *cattleWSProxy) {
	cattleAddr := config.CattleAddr
	director := func(req *http.Request) {
		req.URL.Scheme = "http"
		req.URL.Host = cattleAddr
	}
	cattleProxy := &httputil.ReverseProxy{
		Director:      director,
		FlushInterval: time.Millisecond * 100,
	}

	reverseProxy := &proxyProtocolConverter{
		reverseProxy: cattleProxy,
		httpsPorts:   config.ProxyProtoHTTPSPorts,
	}

	wsProxy := &cattleWSProxy{
		reverseProxy: reverseProxy,
		cattleAddr:   cattleAddr,
	}

	return reverseProxy, wsProxy
}

type proxyProtocolConverter struct {
	reverseProxy *httputil.ReverseProxy
	httpsPorts   map[int]bool
}

func (h *proxyProtocolConverter) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	proxyprotocol.AddHeaders(req, h.httpsPorts)
	h.reverseProxy.ServeHTTP(rw, req)
}

type cattleWSProxy struct {
	reverseProxy *proxyProtocolConverter
	cattleAddr   string
}

func (h *cattleWSProxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if strings.EqualFold(req.Header.Get("Upgrade"), "websocket") {
		proxyprotocol.AddHeaders(req, h.reverseProxy.httpsPorts)
		h.serveWebsocket(rw, req)
	} else {
		h.reverseProxy.ServeHTTP(rw, req)
	}
}

func (h *cattleWSProxy) serveWebsocket(rw http.ResponseWriter, req *http.Request) {
	// Inspired by https://groups.google.com/forum/#!searchin/golang-nuts/httputil.ReverseProxy$20$2B$20websockets/golang-nuts/KBx9pDlvFOc/01vn1qUyVdwJ
	target := h.cattleAddr
	d, err := net.Dial("tcp", target)
	if err != nil {
		log.WithField("error", err).Error("Error dialing websocket backend.")
		http.Error(rw, "Unable to establish websocket connection: can't dial.", 500)
		return
	}
	hj, ok := rw.(http.Hijacker)
	if !ok {
		http.Error(rw, "Unable to establish websocket connection: no hijacker.", 500)
		return
	}
	nc, _, err := hj.Hijack()
	if err != nil {
		log.WithField("error", err).Error("Hijack error.")
		http.Error(rw, "Unable to establish websocket connection: can't hijack.", 500)
		return
	}
	defer nc.Close()
	defer d.Close()

	err = req.Write(d)
	if err != nil {
		log.WithField("error", err).Error("Error copying request to target.")
		return
	}

	errc := make(chan error, 2)
	cp := func(dst io.Writer, src io.Reader) {
		_, err := io.Copy(dst, src)
		errc <- err
	}
	go cp(d, nc)
	go cp(nc, d)
	<-errc
}
