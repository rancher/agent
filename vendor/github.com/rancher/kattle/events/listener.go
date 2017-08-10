package events

import (
	"github.com/Sirupsen/logrus"
	"github.com/rancher/event-subscriber/events"
	"github.com/rancher/kattle/handlers"
)

func Listen(eventURL, accessKey, secretKey string, workerCount int) error {
	logrus.Infof("Listening for events on %v", eventURL)

	eventHandlers := handlers.GetHandlers()
	pingConfig := events.PingConfig{
		SendPingInterval:  5000,
		CheckPongInterval: 5000,
		MaxPongWait:       60000,
	}
	router, err := events.NewEventRouter("", 0, eventURL, accessKey, secretKey, nil, eventHandlers, "", workerCount, pingConfig)
	if err != nil {
		return err
	}

	return router.StartWithoutCreate(nil)
}
