package handlers

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/rancher/agent/handlers/docker"
	"github.com/rancher/agent/handlers/progress"
	"github.com/rancher/agent/handlers/utils"
	revents "github.com/rancher/go-machine-service/events"
	"github.com/rancher/go-rancher/client"
	"golang.org/x/net/context"
	"github.com/rancher/agent/model"
	"github.com/mitchellh/mapstructure"
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
	client := docker.GetClient(utils.DefaultVersion)
	instance := model.Instance{
		UUID: instanceData.UUID,
		AgentID: instanceData.AgentID,
		ExternalID: instanceData.ExternalID,
	}
	container := utils.GetContainer(client, &instance, true)

	if container == nil {
		logrus.Infof("Can not call [%v}, container not exist", instance.UUID)
		return nil
	}

	inspect, _ := client.ContainerInspect(context.Background(), container.ID)
	running := inspect.State.Running
	if !running {
		logrus.Error(fmt.Errorf("Can not call [%v}, container not running", container.ID))
		return nil
	}
	progress := progress.Progress{Request: delegateEvent, Client: cli}
	exitCode, output, data := utils.NsExec(inspect.State.Pid, delegateEvent)
	if exitCode == 0 {
		return replyWithParent(data, delegateEvent, event, cli)
	}
	progress.Update(fmt.Sprintf("Update fail, exitCode %v, output data %v", exitCode, output))
	return nil
}
