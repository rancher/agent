package docker

import (
	"sync"

	"github.com/leodotcloud/log"
	revents "github.com/rancher/event-subscriber/events"
	"github.com/rancher/go-rancher/v2"
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
	log.Info("Compute lock")
	defer func() {
		log.Info("Compute unlock")
		computeLock.Unlock()
	}()

	return f()
}
