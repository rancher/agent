package events

import (
	"github.com/Sirupsen/logrus"
	//"github.com/rancher/agent/handlers"
	"../handlers"
	revents "github.com/rancher/go-machine-service/events"
)

func Listen(eventURL, accessKey, secretKey string, workerCount int) error {
	logrus.Infof("Listening for events on %v", eventURL)

	eventHandlers := handlers.GetHandlers()
	router, err := revents.NewEventRouter("", 0, eventURL, accessKey, secretKey, nil, eventHandlers, "", workerCount)
	if err != nil {
		return err
	}
	err = router.StartWithoutCreate(nil)
	return err
}
