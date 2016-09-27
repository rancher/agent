package events

import (
	"github.com/Sirupsen/logrus"
	"github.com/rancher/agent/handlers"
	//"github.com/rancher/agent/service/cadvisor"
	"github.com/rancher/agent/service/hostapi"
	"github.com/rancher/agent/utilities/config"
	revents "github.com/rancher/event-subscriber/events"
	"os"
	"runtime"
	"time"
)

func Listen(eventURL, accessKey, secretKey string, workerCount int) error {
	logrus.Infof("Listening for events on %v", eventURL)

	config.SetAccessKey(accessKey)
	config.SetSecretKey(secretKey)
	config.SetAPIURL(eventURL)

	config.PhysicalHostUUID(true)

	logrus.Info("launching hostapi")
	go hostapi.StartUp()

	if runtime.GOOS == "linux" {
		//logrus.Info("launching cadvisor")
		//go cadvisor.StartUp()

		go func() {
			timestamps := time.Time{}
			for {
				if !checkTS(&timestamps) {
					logrus.Info("timestamp files have been changed. Exiting go-agent")
					os.Exit(1)
				}
				time.Sleep(time.Duration(2) * time.Second)
			}
		}()
	}

	eventHandlers := handlers.GetHandlers()
	router, err := revents.NewEventRouter("", 0, eventURL, accessKey, secretKey, nil, eventHandlers, "", workerCount, revents.DefaultPingConfig)
	if err != nil {
		return err
	}
	err = router.StartWithoutCreate(nil)
	return err
}

func checkTS(timestamps *time.Time) bool {
	stampFile := config.Stamp()
	stats, err := os.Stat(stampFile)
	if err != nil {
		return true
	}
	ts := stats.ModTime()
	// check whether timestamps has been initialized
	if timestamps.IsZero() {
		*timestamps = ts
	}
	return timestamps.Equal(ts)
}
