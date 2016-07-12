package handlers

import (
	"github.com/Sirupsen/logrus"
	"github.com/rancher/agent/handlers/docker_client"
	"github.com/rancher/agent/handlers/progress"
	"github.com/rancher/agent/handlers/utils"
	"github.com/rancher/agent/model"
	revents "github.com/rancher/go-machine-service/events"
	"github.com/rancher/go-rancher/client"
	"sync"
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

	ins_with_lock := InstanceWithLock{mu: sync.Mutex{}, in: instance}
	ins_with_lock.mu.Lock()
	defer ins_with_lock.mu.Unlock()
	if utils.IsInstanceActive(ins_with_lock.in, host) {
		logrus.Info("instance is activated")
		utils.RecordState(docker_client.GetClient(utils.DEFAULT_VERSION), instance, "")
		return reply(event.Data, utils.GetResponseData(event, event.Data), cli)
	}

	utils.DoInstanceActivate(instance, host, &progress)
	//data := utils.Get_response_data(event, event.Data)

	return reply(nil, event, cli)
}
