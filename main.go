package main

import (
	"github.com/Sirupsen/logrus"
	"github.com/rancher/agent/events"
	"os"
)

func main() {
	logrus.Info("Launching agent")

	url := os.Getenv("CATTLE_URL")
	accessKey := os.Getenv("CATTLE_ACCESS_KEY")
	secretKey := os.Getenv("CATTLE_SECRET_KEY")
	workerCount := 10

	err := events.Listen(url, accessKey, secretKey, workerCount)
	if err != nil {
		logrus.Fatalf("Exiting. Error: %v", err)
	}
}
