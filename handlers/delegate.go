package handlers

import (
	"errors"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/mitchellh/mapstructure"
	"github.com/rancher/agent/handlers/docker"
	"github.com/rancher/agent/handlers/progress"
	"github.com/rancher/agent/handlers/utils"
	"github.com/rancher/agent/model"
	revents "github.com/rancher/go-machine-service/events"
	"github.com/rancher/go-rancher/client"
	"golang.org/x/net/context"
)

func DelegateRequest(event *revents.Event, cli *client.RancherClient) error {
	instanceData := event.Data
	_, ok1 := utils.GetFieldsIfExist(instanceData, "token")
	kind, ok2 := utils.GetFieldsIfExist(instanceData, "kind")
	if !ok2 || kind != "container" || !ok1 {
		return errors.New("delegate operation failed")
	}
	client := docker.GetClient(utils.DefaultVersion)
	var instance model.Instance
	mapstructure.Decode(instanceData, &instance)
	container := utils.GetContainer(client, &instance, true)

	if container == nil {
		logrus.Infof("Can not call [%v}, container not exist", instance.UUID)
		return errors.New("delegate operation failed")
	}

	inspect, _ := client.ContainerInspect(context.Background(), container.ID)
	running := inspect.State.Running
	if !running {
		logrus.Error(fmt.Errorf("Can not call [%v}, container not exist", instance.UUID))
	}
	progress := progress.Progress{}
	exitCode, output, data := utils.NsExec(inspect.State.Pid, event)

	if exitCode == 0 {
		return reply(event.Data, data, cli)
	}
	progress.Update("Update failed" + output)
	return errors.New("delegate operation failed")
}
