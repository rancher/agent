package events

import (
	"github.com/Sirupsen/logrus"
	"github.com/rancher/agent/handlers"
	"github.com/rancher/agent/handlers/cadvisor"
	"github.com/rancher/agent/handlers/hostapi"
	"github.com/rancher/agent/handlers/utils"
	revents "github.com/rancher/go-machine-service/events"
)

func Listen(eventURL, accessKey, secretKey string, workerCount int) error {
	logrus.Infof("Listening for events on %v", eventURL)

	utils.SetAccessKey(accessKey)
	utils.SetSecretKey(secretKey)
	utils.SetAPIURL(eventURL)

	utils.PhysicalHostUUID(true)

	logrus.Info("launching hostapi")
	go hostapi.StartUp()

	logrus.Info("launching cadvisor")
	go cadvisor.StartUp()

	eventHandlers := handlers.GetHandlers()
	router, err := revents.NewEventRouter("", 0, eventURL, accessKey, secretKey, nil, eventHandlers, "", workerCount)
	if err != nil {
		return err
	}
	err = router.StartWithoutCreate(nil)
	return err
}
