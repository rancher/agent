package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/rancher/rancher/pkg/remotedialer"
	"github.com/sirupsen/logrus"
)

const (
	Token  = "X-API-Tunnel-Token"
	ID     = "X-API-Tunnel-ID"
	Params = "X-API-Tunnel-Params"
)

func main() {
	if os.Getenv("CATTLE_DEBUG") == "true" || os.Getenv("RANCHER_DEBUG") == "true" {
		logrus.SetLevel(logrus.DebugLevel)
	}

	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	token := os.Getenv("CATTLE_TOKEN")
	params := map[string]interface{}{
		"customConfig": map[string]interface{}{
			"address":         os.Getenv("CATTLE_ADDRESS"),
			"internalAddress": os.Getenv("CATTLE_INTERNAL_ADDRESS"),
			"roles":           split(os.Getenv("CATTLE_ROLE")),
		},
		"requestedHostname": os.Getenv("CATTLE_NODE_NAME"),
	}

	for k, v := range params {
		if m, ok := v.(map[string]string); ok {
			for k, v := range m {
				logrus.Infof("Option %s=%s", k, v)
			}
		} else {
			logrus.Infof("Option %s=%v", k, v)
		}
	}

	bytes, err := json.Marshal(params)
	if err != nil {
		return err
	}

	headers := map[string][]string{
		Token:  {token},
		Params: {base64.StdEncoding.EncodeToString(bytes)},
	}

	server := os.Getenv("CATTLE_SERVER")
	serverURL, err := url.Parse(server)
	if err != nil {
		return err
	}

	wsURL := fmt.Sprintf("wss://%s/v3/connect", serverURL.Host)
	remotedialer.ClientConnect(wsURL, http.Header(headers), nil, func(proto, address string) bool {
		switch proto {
		case "tcp":
			return true
		case "unix":
			return address == "/var/run/docker.sock"
		}
		return false
	})

	return nil
}

func split(s string) []string {
	var result []string
	for _, part := range strings.Split(s, ",") {
		p := strings.TrimSpace(part)
		if p != "" {
			result = append(result, p)
		}
	}
	if len(result) == 1 && result[0] == "" {
		return nil
	}
	return result
}
