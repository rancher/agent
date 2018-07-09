package main

import (
	"github.com/rancher/agent/tests/framework"
	"github.com/rancher/log"
	"net/http"
)

func main() {
	log.Infof("Starting test event server.")
	s := framework.NewServer()
	handler := http.Handler(s)
	err := http.ListenAndServe("localhost:8089", handler)
	if err == nil {
		log.Infof("Test event server exited.")
	} else {
		log.Errorf("Test event server errored out: %v", err)
	}
}
