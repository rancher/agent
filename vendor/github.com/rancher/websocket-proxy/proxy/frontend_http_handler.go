package proxy

import (
	"bufio"
	"errors"
	"io"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/dgrijalva/jwt-go"

	"github.com/rancher/websocket-proxy/proxy/proxyprotocol"
)

type FrontendHTTPHandler struct {
	FrontendHandler
	HTTPSPorts  map[int]bool
	TokenLookup *TokenLookup
}

func (h *FrontendHTTPHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if err := h.serveHTTP(rw, req); err != nil {
		log.Errorf("Failed to handle %s %s: %v", req.Method, req.URL.String(), err)
		rw.WriteHeader(500)
		rw.Write([]byte(err.Error()))
	}
}

func (h *FrontendHTTPHandler) serveHTTP(rw http.ResponseWriter, req *http.Request) error {
	token, hostKey, authed, err := h.authAndLookup(req)
	if err != nil {
		http.Error(rw, "Service Unavailable", 503)
		return nil
	}
	if !authed {
		http.Error(rw, "Failed authentication", 401)
		return nil
	}

	data, _ := token.Claims["proxy"].(map[string]interface{})
	address, _ := data["address"].(string)
	scheme, _ := data["scheme"].(string)

	proxyprotocol.AddHeaders(req, h.HTTPSPorts)
	proxyprotocol.AddForwardedFor(req)

	reader, writer, err := NewHTTPPipe(rw, h.backend, hostKey)
	if err != nil {
		log.Errorf("Failed to construct pipe to backend %s: %v", hostKey, err)
		return err
	}
	defer writer.Close()
	defer reader.Close()

	hijack := h.shouldHijack(req)

	if err := writer.WriteRequest(req, hijack, address, scheme); err != nil {
		log.Errorf("Failed to write request to backend: %v", err)
		return err
	}

	var input io.Reader
	var output io.Writer

	if hijack {
		hijacker, ok := rw.(http.Hijacker)
		if !ok {
			return errors.New("Invalid input")
		}

		httpConn, buf, err := hijacker.Hijack()
		if err != nil {
			log.Errorf("Failed to hijack connection: %v", err)
			return err
		}
		defer httpConn.Close()
		defer buf.Flush()

		input = buf
		output = buf
	} else {
		input = req.Body
		output = rw
	}

	go func() {
		io.Copy(writer, input)
		writer.Close()
	}()
	_, err = io.Copy(flusher{output}, reader)
	return err
}

type flusher struct {
	writer io.Writer
}

func (f flusher) Write(b []byte) (int, error) {
	defer flush(f.writer)
	return f.writer.Write(b)
}

func flush(writer io.Writer) {
	if buf, ok := writer.(*bufio.ReadWriter); ok {
		buf.Flush()
	} else if buf, ok := writer.(http.Flusher); ok {
		buf.Flush()
	}
}

func (h *FrontendHTTPHandler) shouldHijack(req *http.Request) bool {
	return req.Header.Get("Connection") == "Upgrade"
}

func (h *FrontendHTTPHandler) authAndLookup(req *http.Request) (*jwt.Token, string, bool, error) {
	token, hostKey, authErr := h.FrontendHandler.auth(req)
	if authErr == nil {
		return token, hostKey, true, nil
	} else if !IsNoTokenError(authErr) {
		log.Infof("Frontend auth failed: %v. Getting new token.", authErr)
	}

	tokenString, err := h.TokenLookup.Lookup(req)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Error looking up token.")
		return nil, "", false, err
	}

	token, err = jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return h.parsedPublicKey, nil
	})
	if err != nil {
		return nil, "", false, err
	}

	if !token.Valid {
		return nil, "", false, nil
	}

	hostUUID, found := token.Claims["hostUuid"]
	if found {
		if hostKey, ok := hostUUID.(string); ok && h.backend.hasBackend(hostKey) {
			return token, hostKey, true, nil
		}
	}
	log.WithFields(log.Fields{"hostUuid": hostUUID}).Infof("Invalid backend host requested.")
	return nil, "", false, nil
}
