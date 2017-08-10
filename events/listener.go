package events

import (
	"os"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/agent/handlers"
	"github.com/rancher/agent/service/hostapi"
	"github.com/rancher/agent/utils"
	revents "github.com/rancher/event-subscriber/events"
	kattleHandlers "github.com/rancher/kattle/handlers"
	"github.com/rancher/kattle/kubernetesclient"
	"github.com/rancher/kattle/watch"
)

func Listen(eventURL, accessKey, secretKey string, workerCount int, kubernetes string) error {
	logrus.Infof("Listening for events on %v", eventURL)

	utils.SetAccessKey(accessKey)
	utils.SetSecretKey(secretKey)
	utils.SetAPIURL(eventURL)

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
		return err
	}

	if kubernetes == "import" {
		clientset, err := kubernetesclient.CreateKubernetesClient()
		if err != nil {
			return err
		}

		version, err := clientset.Discovery().ServerVersion()
		if err != nil {
			return err
		}
		logrus.Infof("Kubernetes version: %s", version.String())

		watchClient := watch.NewClient(clientset)
		watchClient.Start()

		time.Sleep(5 * time.Second)

		kattleHandlers.WatchClient = watchClient
		kattleHandlers.Clientset = clientset

		for name, handler := range kattleHandlers.GetHandlers() {
			eventHandlers[name] = handler
		}
	}

	pingConfig := revents.PingConfig{
		SendPingInterval:  5000,
		CheckPongInterval: 5000,
		MaxPongWait:       60000,
	}
	router, err := revents.NewEventRouter("", 0, eventURL, accessKey, secretKey, nil, eventHandlers, "", workerCount, pingConfig)
	if err != nil {
		return err
	}
	err = router.StartWithoutCreate(nil)
	return err
}

func checkTS(timestamps *time.Time) bool {
	stampFile := utils.Stamp()
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
