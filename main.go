package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/rancher/agent/cloudprovider"
	"github.com/rancher/agent/events"
	"github.com/rancher/agent/register"
	"github.com/rancher/agent/utilities/config"
	"github.com/rancher/log"
	logserver "github.com/rancher/log/server"

	_ "github.com/rancher/agent/cloudprovider/aliyun"
	_ "github.com/rancher/agent/cloudprovider/aws"
)

var (
	VERSION = "dev"
)

func main() {
	logserver.StartServerWithDefaults()
	version := flag.Bool("version", false, "go-agent version")
	rurl := flag.String("url", "", "registration url")
	registerService := flag.String("register-service", "", "register rancher-agent service")
	unregisterService := flag.Bool("unregister-service", false, "unregister rancher-agent service")
	flag.Parse()
	if *version {
		fmt.Printf("go-agent version %s \n", VERSION)
		os.Exit(0)
	}

	if os.Getenv("CATTLE_SCRIPT_DEBUG") != "" || os.Getenv("RANCHER_DEBUG") != "" {
		log.SetLevelString("debug")
	}

	if err := register.Init(*registerService, *unregisterService); err != nil {
		log.Fatalf("Failed to Initialize Service err: %v", err)
	}

	if *rurl != "" {
		err := register.RunRegistration(*rurl)
		if err != nil {
			log.Errorf("registration failed. err: %v", err)
			os.Exit(1)
		}
	}

	log.Info("Launching agent")

	url := os.Getenv("CATTLE_URL")
	accessKey := os.Getenv("CATTLE_ACCESS_KEY")
	secretKey := os.Getenv("CATTLE_SECRET_KEY")
	workerCount := 250

	if config.DetectCloudProvider() {
		log.Info("Detecting cloud provider")
		cloudprovider.GetCloudProviderInfo()
	}

	err := events.Listen(url, accessKey, secretKey, workerCount)
	if err != nil {
		log.Fatalf("Exiting. Error: %v", err)
		register.NotifyShutdown(err)
	}
}
