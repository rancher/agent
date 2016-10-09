package main

import (
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/agent/events"
	"github.com/rancher/agent/utilities/config"
	"path"
)

func main() {
	//setup log config
	logFile := config.LogFile()
	err := os.MkdirAll(path.Dir(logFile), 0666)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Info("Can not make a directory. Exiting go-agent")
		os.Exit(1)
	}
	file, err := os.Create(logFile)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Info("Can not create log file. Exiting go-agent")
		os.Exit(1)
	}
	defer file.Close()
	logrus.SetOutput(file)

	logrus.Info("Launching agent")

	url := os.Getenv("CATTLE_URL")
	accessKey := os.Getenv("CATTLE_ACCESS_KEY")
	secretKey := os.Getenv("CATTLE_SECRET_KEY")
	workerCount := 50

	err = events.Listen(url, accessKey, secretKey, workerCount)
	if err != nil {
		logrus.Fatalf("Exiting. Error: %v", err)
	}
}
