package handlers

import (
	"github.com/Sirupsen/logrus"
	revents "github.com/rancher/go-machine-service/events"
	"github.com/rancher/go-rancher/client"
	"sync"
	"github.com/rancher/agent/handlers/progress"
	"github.com/rancher/agent/handlers/utils"
	"github.com/rancher/agent/handlers/docker_client"
	"github.com/rancher/agent/model"
)

type InstanceWithLock struct {
	mu sync.Mutex
	in *model.Instance
}

func InstanceActivate(event *revents.Event, cli *client.RancherClient) error {
	logrus.Infof("Received event: Name: %s, Event Id: %s, Resource Id: %s", event.Name, event.ID, event.ResourceID)
	instance, host := utils.GetInstanceAndHost(event)

	progress := progress.Progress{}

	if instance != nil {
		processData, ok := event.Data["processData"]
		if !ok {
			instance.ProcessData = processData
		}
	}

	ins_with_lock := InstanceWithLock{mu:sync.Mutex{}, in: instance}
	ins_with_lock.mu.Lock()
	defer ins_with_lock.mu.Unlock()
	if utils.Is_instance_active(ins_with_lock.in, host) {
		utils.Record_state(docker_client.Get_client(utils.DEFAULT_VERSION), instance, "")
		return reply(event.Data, utils.Get_response_data(event, event.Data), cli)
	}


	utils.Do_instance_activate(instance, host, &progress)
	//data := utils.Get_response_data(event, event.Data)

	return reply(nil, event, cli)
}
