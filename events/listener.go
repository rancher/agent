package events

import (
	"os"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"github.com/rancher/agent/handlers"
	"github.com/rancher/agent/service/hostapi"
	"github.com/rancher/agent/utilities/config"
	revents "github.com/rancher/event-subscriber/events"
)

func Listen(eventURL, accessKey, secretKey string, workerCount int) error {
	logrus.Infof("Listening for events on %v", eventURL)

	config.SetAccessKey(accessKey)
	config.SetSecretKey(secretKey)
	config.SetAPIURL(eventURL)

	config.PhysicalHostUUID(true)
	config.SetDockerUUID()

	logrus.Info("launching hostapi")
	go hostapi.StartUp()

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

	eventHandlers, err := handlers.GetHandlers()
	if err != nil {
		return errors.Wrap(err, "Failed to get event handlers")
	}

	pingConfig := revents.PingConfig{
		SendPingInterval:  5000,
		CheckPongInterval: 5000,
		MaxPongWait:       60000,
	}
	router, err := revents.NewEventRouter("", 0, eventURL, accessKey, secretKey, nil, eventHandlers, "", workerCount, pingConfig)
	if err != nil {
		return errors.Wrap(err, "Failed to create new event router")
	}
	err = router.StartWithoutCreate(nil)
	if err != nil {
		return errors.Wrap(err, "Error encountered while running event router")
	}
	return nil
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
