package handlers

import (
	"github.com/Sirupsen/logrus"
	revents "github.com/rancher/go-machine-service/events"
	"github.com/rancher/go-rancher/client"
	"sync"
	"github.com/docker/engine-api/client"
)

type InstanceWithLock struct {
	mu sync.Mutex
	in *map[string]interface{}
}

func InstanceActivate(event *revents.Event, cli *client.RancherClient) error {
	logrus.Infof("Received event: Name: %s, Event Id: %s, Resource Id: %s", event.Name, event.ID, event.ResourceID)
	instance, host := getInstanceAndHost(event)

	progess := Progress(event)

	if instance != nil {
		processData, ok := event.Data["processData"]
		if ok != nil {
			instance["processData"] = processData
		}
	}

	ins_with_lock := InstanceWithLock{mu:&sync.Mutex{}, in: &instance}
	ins_with_lock.mu.Lock()
	if is_instance_active(ins_with_lock.in, host) {
		record_state(client, instance)
		return reply(event, get_response_data(event, event.Data), cli)
	}


	do_instance_activate(instance, host, progress)
	data := get_response_data(event, event.Data)

	return reply(nil, event, cli)
}
