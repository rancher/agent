package handlers

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/mitchellh/mapstructure"
	"github.com/rancher/agent/core/delegate"
	"github.com/rancher/agent/core/progress"
	"github.com/rancher/agent/model"
	"github.com/rancher/agent/utilities/constants"
	"github.com/rancher/agent/utilities/docker"
	"github.com/rancher/agent/utilities/utils"
	revents "github.com/rancher/event-subscriber/events"
	"github.com/rancher/go-rancher/client"
	"golang.org/x/net/context"
)

func DelegateRequest(event *revents.Event, cli *client.RancherClient) error {
	indata, _ := utils.GetFieldsIfExist(event.Data, "instanceData")
	deEvent, _ := utils.GetFieldsIfExist(event.Data, "event")
	var instanceData model.InstanceData
	mapstructure.Decode(utils.InterfaceToMap(indata), &instanceData)
	var delegateEvent *revents.Event
	mapstructure.Decode(utils.InterfaceToMap(deEvent), &delegateEvent)
	if instanceData.Kind != "container" || instanceData.Token == "" {
		return nil
	}
	client := docker.GetClient(constants.DefaultVersion)
	instance := model.Instance{
		UUID:       instanceData.UUID,
		AgentID:    instanceData.AgentID,
		ExternalID: instanceData.ExternalID,
	}
	container := utils.GetContainer(client, &instance, true)

	if container == nil {
		logrus.Infof("Can not call [%v], container not exist", instance.UUID)
		return nil
	}

	inspect, _ := client.ContainerInspect(context.Background(), container.ID)
	running := inspect.State.Running
	if !running {
		logrus.Error(fmt.Errorf("Can not call [%v], container not running", container.ID))
		return nil
	}
	progress := progress.Progress{Request: delegateEvent, Client: cli}
	exitCode, output, data := delegate.NsExec(inspect.State.Pid, delegateEvent)
	if exitCode == 0 {
		return replyWithParent(data, delegateEvent, event, cli)
	}
	progress.Update(fmt.Sprintf("Update fail, exitCode %v, output data %v", exitCode, output))
	return nil
}
