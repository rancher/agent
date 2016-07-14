package main

import (
	"github.com/Sirupsen/logrus"
	"github.com/rancher/agent/tests/framework"
	"net/http"
)

func main() {
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	logrus.Infof("Starting test event server.")
	s := framework.NewServer()
	handler := http.Handler(s)
	err := http.ListenAndServe("localhost:8089", handler)
	if err == nil {
		logrus.Infof("Test event server exited.")
	} else {
		logrus.Errorf("Test event server errored out: %v", err)
	}
}
