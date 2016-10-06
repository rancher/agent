package main

import (
	"flag"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/rancher/agent/events"
	"os"
)

var VERSION = "dev"

func main() {
	version := flag.Bool("version", false, "go-agent version")
	flag.Parse()
	if *version {
		fmt.Printf("go-agent version %s \n", VERSION)
		os.Exit(0)
	}

	logrus.Info("Launching agent")

	url := os.Getenv("CATTLE_URL")
	accessKey := os.Getenv("CATTLE_ACCESS_KEY")
	secretKey := os.Getenv("CATTLE_SECRET_KEY")
	workerCount := 50

	err := events.Listen(url, accessKey, secretKey, workerCount)
	if err != nil {
		logrus.Fatalf("Exiting. Error: %v", err)
	}
}
