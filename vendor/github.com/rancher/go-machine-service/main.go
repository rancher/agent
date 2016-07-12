package main

import (
	"flag"
	"fmt"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/rancher/go-machine-service/dynamic"
	"github.com/rancher/go-machine-service/events"
	"github.com/rancher/go-machine-service/handlers"
)

var (
	GITCOMMIT = "HEAD"
)

func main() {
	processCmdLineFlags()

	log.WithField("gitcommit", GITCOMMIT).Info("Starting go-machine-service...")

	apiURL := os.Getenv("CATTLE_URL")
	accessKey := os.Getenv("CATTLE_ACCESS_KEY")
	secretKey := os.Getenv("CATTLE_SECRET_KEY")

	ready := make(chan bool, 2)
	done := make(chan error)

	go func() {
		eventHandlers := map[string]events.EventHandler{
			"machinedriver.reactivate": handlers.ActivateDriver,
			"machinedriver.activate":   handlers.ActivateDriver,
			"machinedriver.update":     handlers.ActivateDriver,
			"machinedriver.deactivate": handlers.DeactivateDriver,
			"machinedriver.remove":     handlers.RemoveDriver,
			"ping":                     handlers.PingNoOp,
		}

		router, err := events.NewEventRouter("goMachineService-machine", 2000, apiURL, accessKey, secretKey,
			nil, eventHandlers, "machineDriver", 10)
		if err == nil {
			err = router.Start(ready)
		}
		done <- err
	}()

	go func() {
		eventHandlers := map[string]events.EventHandler{
			"physicalhost.create":    handlers.CreateMachine,
			"physicalhost.bootstrap": handlers.ActivateMachine,
			"physicalhost.remove":    handlers.PurgeMachine,
			"ping":                   handlers.PingNoOp,
		}

		router, err := events.NewEventRouter("goMachineService", 2000, apiURL, accessKey, secretKey,
			nil, eventHandlers, "physicalhost", 10)
		if err == nil {
			err = router.Start(ready)
		}
		done <- err
	}()

	go func() {
		log.Infof("Waiting for handler registration (1/2)")
		<-ready
		log.Infof("Waiting for handler registration (2/2)")
		<-ready
		if err := dynamic.DownloadAllDrivers(); err != nil {
			log.Fatalf("Error updating drivers: %v", err)
		}
	}()

	err := <-done
	if err == nil {
		log.Infof("Exiting go-machine-service")
	} else {
		log.Fatalf("Exiting go-machine-service: %v", err)
	}
}

func processCmdLineFlags() {
	// Define command line flags
	logLevel := flag.String("loglevel", "info", "Set the default loglevel (default:info) [debug|info|warn|error]")
	version := flag.Bool("v", false, "read the version of the go-machine-service")
	output := flag.String("o", "", "set the output file to write logs into, default is stdout")

	flag.Parse()

	if *output != "" {
		var f *os.File
		if _, err := os.Stat(*output); os.IsNotExist(err) {
			f, err = os.Create(*output)
			if err != nil {
				fmt.Printf("could not create file=%s for logging, err=%v\n", *output, err)
				os.Exit(1)
			}
		} else {
			var err error
			f, err = os.OpenFile(*output, os.O_RDWR|os.O_APPEND, 0)
			if err != nil {
				fmt.Printf("could not open file=%s for writing, err=%v\n", *output, err)
				os.Exit(1)
			}
		}
		log.SetFormatter(&log.JSONFormatter{})
		log.SetOutput(f)
	}

	if *version {
		fmt.Printf("go-machine-service\t gitcommit=%s\n", GITCOMMIT)
		os.Exit(0)
	}

	// Process log level.  If an invalid level is passed in, we simply default to info.
	if parsedLogLevel, err := log.ParseLevel(*logLevel); err == nil {
		log.WithFields(log.Fields{
			"logLevel": *logLevel,
		}).Info("Setting log level")
		log.SetLevel(parsedLogLevel)
	}
}
