package events

import (
	"github.com/Sirupsen/logrus"
	"github.com/rancher/agent/handlers"
	revents "github.com/rancher/go-machine-service/events"
	"github.com/rancher/agent/handlers/cadvisor"
	"github.com/rancher/agent/handlers/hostapi"
	"github.com/rancher/agent/handlers/utils"
)

func Listen(eventURL, accessKey, secretKey string, workerCount int) error {
	logrus.Infof("Listening for events on %v", eventURL)

	utils.SetAccessKey(accessKey)
	utils.SetSecretKey(secretKey)
	utils.SetAPIURL(eventURL)

	utils.PhysicalHostUUID(true)

	logrus.Info("launching hostapi")
	go hostapi.HostAPIStartUp()

	logrus.Info("launching cadvisor")
	go cadvisor.CadvisorStartUp()

	eventHandlers := handlers.GetHandlers()
	router, err := revents.NewEventRouter("", 0, eventURL, accessKey, secretKey, nil, eventHandlers, "", workerCount)
	if err != nil {
		return err
	}
	err = router.StartWithoutCreate(nil)
	return err
}
