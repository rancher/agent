package handlers

import (
	"github.com/Sirupsen/logrus"
	revents "github.com/rancher/go-machine-service/events"
	"github.com/rancher/go-rancher/client"
	"sync"
	"../handlers/progress"
	"../handlers/utils"
	"../handlers/docker_client"
)

type InstanceWithLock struct {
	mu sync.Mutex
	in *map[string]interface{}
}

func InstanceActivate(event *revents.Event, cli *client.RancherClient) error {
	logrus.Infof("Received event: Name: %s, Event Id: %s, Resource Id: %s", event.Name, event.ID, event.ResourceID)
	instance, host := utils.GetInstanceAndHost(event)

	progress := progress.Progress{}

	if instance != nil {
		processData, ok := event.Data["processData"]
		if ok != nil {
			instance["processData"] = processData
		}
	}

	ins_with_lock := InstanceWithLock{mu:&sync.Mutex{}, in: &instance}
	ins_with_lock.mu.Lock()
	defer ins_with_lock.mu.Unlock()
	if utils.Is_instance_active(&ins_with_lock.in, host) {
		utils.Record_state(docker_client.Get_client(utils.DEFAULT_VERSION), instance, nil)
		return reply(event, utils.Get_response_data(event, event.Data), cli)
	}


	utils.Do_instance_activate(instance, host, &progress)
	//data := utils.Get_response_data(event, event.Data)

	return reply(nil, event, cli)
}
