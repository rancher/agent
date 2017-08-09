package utils

import (
	"github.com/Sirupsen/logrus"
	revents "github.com/rancher/event-subscriber/events"
	"github.com/rancher/go-rancher/v3"
	"sync"
)

var (
	SerializeCompute = false
	computeLock      = sync.Mutex{}
)

func SerializeHandler(f revents.EventHandler) revents.EventHandler {
	return func(event *revents.Event, cli *client.RancherClient) error {
		return Serialize(func() error {
			return f(event, cli)
		})
	}
}

func Serialize(f func() error) error {
	if !SerializeCompute {
		return f()
	}

	computeLock.Lock()
	logrus.Info("Compute lock")
	defer func() {
		logrus.Info("Compute unlock")
		computeLock.Unlock()
	}()

	return f()
}
