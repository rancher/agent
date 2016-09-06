package events

import (
	"github.com/Sirupsen/logrus"
	"github.com/rancher/agent/handlers"
	"github.com/rancher/agent/service/cadvisor"
	"github.com/rancher/agent/service/hostapi"
	"github.com/rancher/agent/utilities/config"
	revents "github.com/rancher/event-subscriber/events"
	"runtime"
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
		logrus.Info("launching cadvisor")
		go cadvisor.StartUp()
	}

	eventHandlers := handlers.GetHandlers()
	router, err := revents.NewEventRouter("", 0, eventURL, accessKey, secretKey, nil, eventHandlers, "", workerCount, revents.DefaultPingConfig)
	if err != nil {
		return err
	}
	err = router.StartWithoutCreate(nil)
	return err
}
