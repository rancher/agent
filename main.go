package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/agent/cloudprovider"
	"github.com/rancher/agent/events"
	"github.com/rancher/agent/register"
)

var (
	VERSION = "dev"
)

func main() {
	version := flag.Bool("version", false, "go-agent version")
	rurl := flag.String("url", "", "registration url")
	registerService := flag.String("register-service", "", "register rancher-agent service")
	unregisterService := flag.Bool("unregister-service", false, "unregister rancher-agent service")
	flag.Parse()
	if *version {
		fmt.Printf("go-agent version %s \n", VERSION)
		os.Exit(0)
	}
	if runtime.GOOS != "windows" {
		logrus.SetOutput(os.Stdout)
	}

	if os.Getenv("CATTLE_SCRIPT_DEBUG") != "" {
		logrus.SetLevel(logrus.DebugLevel)
	}

	if err := register.Init(*registerService, *unregisterService); err != nil {
		logrus.Fatalf("Failed to Initialize Service err: %v", err)
	}

	if *rurl != "" {
		err := register.RunRegistration(*rurl)
		if err != nil {
			logrus.Errorf("registration failed. err: %v", err)
			os.Exit(1)
		}
	}

	logrus.Info("Launching agent")

	url := os.Getenv("CATTLE_URL")
	accessKey := os.Getenv("CATTLE_ACCESS_KEY")
	secretKey := os.Getenv("CATTLE_SECRET_KEY")
	logrus.Info(url, accessKey, secretKey)
	workerCount := 250

	go cloudprovider.GetCloudProviderInfo()
	err := events.Listen(url, accessKey, secretKey, workerCount)
	if err != nil {
		logrus.Fatalf("Exiting. Error: %v", err)
		register.NotifyShutdown(err)
	}
}
