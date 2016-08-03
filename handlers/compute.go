package handlers

import (
	"encoding/json"
	"github.com/Sirupsen/logrus"
	"github.com/mitchellh/mapstructure"
	"github.com/rancher/agent/handlers/docker"
	"github.com/rancher/agent/handlers/progress"
	"github.com/rancher/agent/handlers/utils"
	"github.com/rancher/agent/model"
	revents "github.com/rancher/go-machine-service/events"
	"github.com/rancher/go-rancher/client"
)

func InstanceActivate(event *revents.Event, cli *client.RancherClient) error {
	logrus.Infof("Received event: Name: %s, Event Id: %s, Resource Id: %s", event.Name, event.ID, event.ResourceID)
	instance, host := utils.GetInstanceAndHost(event)

	progress := progress.Progress{Request: event, Client: cli}

	if processData, ok := event.Data["processData"]; ok && instance != nil {
		instance.ProcessData = processData.(map[string]interface{})
	}
	if utils.IsNoOp(event.Data) {
		utils.RecordState(docker.GetClient(utils.DefaultVersion), instance, "")
		return reply(utils.GetResponseData(event), event, cli)
	}

	if utils.IsInstanceActive(instance, host) {
		logrus.Info("instance is activated")
		utils.RecordState(docker.GetClient(utils.DefaultVersion), instance, "")
		return reply(utils.GetResponseData(event), event, cli)
	}

	utils.DoInstanceActivate(instance, host, &progress)
	return reply(utils.GetResponseData(event), event, cli)
}

func InstanceDeactivate(event *revents.Event, cli *client.RancherClient) error {
	logrus.Infof("Received event: Name: %s, Event Id: %s, Resource Id: %s", event.Name, event.ID, event.ResourceID)
	instance, _ := utils.GetInstanceAndHost(event)

	progress := progress.Progress{Request: event, Client: cli}

	if processData, ok := event.Data["processData"]; ok && instance != nil {
		instance.ProcessData = processData.(map[string]interface{})
	}
	if utils.IsNoOp(event.Data) {
		utils.RecordState(docker.GetClient(utils.DefaultVersion), instance, "")
		return reply(utils.GetResponseData(event), event, cli)
	}
	if utils.IsInstanceInactive(instance) {
		return reply(utils.GetResponseData(event), event, cli)
	}

	timeout, ok := utils.GetFieldsIfExist(event.Data, "processData", "timeout")
	if !ok {
		timeout = 0
	}
	switch timeout.(type) {
	case float64:
		timeout = int(timeout.(float64))
	}
	err := utils.DoInstanceDeactivate(instance, &progress, timeout.(int))

	if err != nil {
		logrus.Error(err)
		return err
	}

	return reply(utils.GetResponseData(event), event, cli)
}

func InstanceForceStop(event *revents.Event, cli *client.RancherClient) error {
	logrus.Infof("Received event: Name: %s, Event Id: %s, Resource Id: %s", event.Name, event.ID, event.ResourceID)
	var request model.InstanceForceStop
	mapstructure.Decode(event.Data["instanceForceStop"], &request)
	return utils.DoInstanceForceStop(&request)
}

func InstanceInspect(event *revents.Event, cli *client.RancherClient) error {
	logrus.Infof("Received event: Name: %s, Event Id: %s, Resource Id: %s", event.Name, event.ID, event.ResourceID)
	var inspect model.InstanceInspect
	mapstructure.Decode(event.Data["instanceInspect"], &inspect)
	inspectResp, _ := utils.DoInstanceInspect(&inspect)
	var inspectJSON map[string]interface{}
	data, err1 := json.Marshal(inspectResp)
	if err1 != nil {
		logrus.Error(err1)
		return err1
	}
	err2 := json.Unmarshal(data, &inspectJSON)
	if err2 != nil {
		logrus.Error(err2)
		return err2
	}
	result := map[string]interface{}{event.ResourceType: inspectJSON}
	return reply(result, event, cli)
}

func InstancePull(event *revents.Event, cli *client.RancherClient) error {
	progress := progress.Progress{Request: event, Client: cli}
	var instancePull model.InstancePull
	mapstructure.Decode(event.Data["instancePull"], &instancePull)
	imageParams := model.ImageParams{Image: instancePull.Image,
		Mode: instancePull.Mode, Complete: instancePull.Complete, Tag: instancePull.Tag}
	imagePull, pullErr := utils.DoInstancePull(&imageParams, &progress)
	if pullErr != nil {
		logrus.Error(pullErr)
	}
	result := map[string]interface{}{}
	if imagePull.ID != "" {
		var imagePullJSON map[string]interface{}
		data, _ := json.Marshal(imagePull)
		json.Unmarshal(data, &imagePullJSON)
		result["fields"] = map[string]interface{}{}
		result["fields"].(map[string]interface{})["dockerImage"] = imagePull
	}
	return reply(utils.GetResponseData(event), event, cli)
}

func InstanceRemove(event *revents.Event, cli *client.RancherClient) error {
	instance, _ := utils.GetInstanceAndHost(event)

	progress := progress.Progress{Request: event, Client: cli}

	if instance != nil && event.Data["processData"] != nil {
		instance.ProcessData = event.Data["processData"].(map[string]interface{})
	}

	if utils.IsInstanceRemoved(instance) {
		return reply(map[string]interface{}{}, event, cli)
	}

	if utils.IsInstanceRemoved(instance) {
		return reply(map[string]interface{}{}, event, cli)
	}
	err := utils.DoInstanceRemove(instance, &progress)
	if err != nil {
		logrus.Error(err)
	}
	return reply(map[string]interface{}{}, event, cli)
}
