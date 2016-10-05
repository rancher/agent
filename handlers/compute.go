package handlers

import (
	"strings"

	"github.com/Sirupsen/logrus"
	engineCli "github.com/docker/engine-api/client"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/rancher/agent/core/compute"
	"github.com/rancher/agent/core/marshaller"
	"github.com/rancher/agent/model"
	"github.com/rancher/agent/utilities/utils"
	revents "github.com/rancher/event-subscriber/events"
	"github.com/rancher/go-rancher/v2"
)

type ComputeHandler struct {
	dockerClient *engineCli.Client
	infoData     model.InfoData
}

func (h *ComputeHandler) InstanceActivate(event *revents.Event, cli *client.RancherClient) error {
	logrus.Infof("Received event: Name: %s, Event Id: %s, Resource Id: %s", event.Name, event.ID, event.ResourceID)
	instance, host, err := utils.GetInstanceAndHost(event)

	if err != nil {
		return errors.WithStack(err)
	}

	progress := utils.GetProgress(event, cli)

	if noOp, ok := utils.GetFieldsIfExist(event.Data, "processData", "containerNoOpEvent"); ok {
		instance.ProcessData.ContainerNoOpEvent = utils.InterfaceToBool(noOp)
	}

	if ok, err := compute.IsInstanceActive(instance, host, h.dockerClient); ok {
		if err := compute.RecordState(h.dockerClient, instance, ""); err != nil {
			return errors.WithStack(err)
		}
		return h.reply(event, cli)
	} else if err != nil {
		return errors.WithStack(err)
	}

	if err := compute.DoInstanceActivate(instance, host, progress, h.dockerClient, h.infoData); err != nil {
		return errors.WithStack(err)
	}
	return h.reply(event, cli)
}

func (h *ComputeHandler) InstanceDeactivate(event *revents.Event, cli *client.RancherClient) error {
	logrus.Infof("Received event: Name: %s, Event Id: %s, Resource Id: %s", event.Name, event.ID, event.ResourceID)
	instance, _, err := utils.GetInstanceAndHost(event)
	if err != nil {
		return errors.WithStack(err)
	}

	if noOp, ok := utils.GetFieldsIfExist(event.Data, "processData", "containerNoOpEvent"); ok {
		instance.ProcessData.ContainerNoOpEvent = utils.InterfaceToBool(noOp)
	}

	if ok, err := compute.IsInstanceInactive(instance, h.dockerClient); err != nil {
		return errors.WithStack(err)
	} else if ok {
		return h.reply(event, cli)
	}

	timeout, ok := utils.GetFieldsIfExist(event.Data, "processData", "timeout")
	if !ok {
		timeout = 10
	}
	switch timeout.(type) {
	case float64:
		timeout = int(timeout.(float64))
	}
	err = compute.DoInstanceDeactivate(instance, h.dockerClient, timeout.(int))
	if err != nil {
		return errors.WithStack(err)
	}

	return h.reply(event, cli)
}

func (h *ComputeHandler) InstanceForceStop(event *revents.Event, cli *client.RancherClient) error {
	logrus.Infof("Received event: Name: %s, Event Id: %s, Resource Id: %s", event.Name, event.ID, event.ResourceID)
	var request model.InstanceForceStop
	err := mapstructure.Decode(event.Data["instanceForceStop"], &request)
	if err != nil {
		return errors.WithStack(err)
	}
	err = compute.DoInstanceForceStop(request, h.dockerClient)
	if err != nil {
		return errors.WithStack(err)
	}
	logrus.Infof("rancher id [%v]: Container with docker id [%v] has been deactivated", event.ResourceID, request.ID)
	return nil
}

func (h *ComputeHandler) InstanceInspect(event *revents.Event, cli *client.RancherClient) error {
	logrus.Infof("Received event: Name: %s, Event Id: %s, Resource Id: %s", event.Name, event.ID, event.ResourceID)
	var inspect model.InstanceInspect
	if err := mapstructure.Decode(event.Data["instanceInspect"], &inspect); err != nil {
		return errors.WithStack(err)
	}
	inspectResp, err := compute.DoInstanceInspect(inspect, h.dockerClient)
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return errors.WithStack(err)
	}
	logrus.Infof("rancher id [%v]: Container with docker id [%v] has been inspected", event.ResourceID, inspect.ID)
	inspectJSON, err := marshaller.StructToMap(inspectResp)
	if err != nil {
		return errors.WithStack(err)
	}
	result := map[string]interface{}{event.ResourceType: inspectJSON}
	return reply(result, event, cli)
}

func (h *ComputeHandler) InstancePull(event *revents.Event, cli *client.RancherClient) error {
	progress := utils.GetProgress(event, cli)
	var instancePull model.InstancePull
	err := mapstructure.Decode(event.Data["instancePull"], &instancePull)
	if err != nil {
		return errors.WithStack(err)
	}
	imageParams := model.ImageParams{
		Image:     instancePull.Image,
		Mode:      instancePull.Mode,
		Complete:  instancePull.Complete,
		Tag:       instancePull.Tag,
		ImageUUID: instancePull.Image.Data.DockerImage.FullName,
	}

	inspect, pullErr := compute.DoInstancePull(imageParams, progress, h.dockerClient)
	if pullErr != nil {
		return errors.WithStack(err)
	}
	logrus.Info("rancher id [%v]: Image with docker id [%v] has been pulled", event.ResourceID, inspect.ID)
	return h.reply(event, cli)
}

func (h *ComputeHandler) InstanceRemove(event *revents.Event, cli *client.RancherClient) error {
	instance, _, err := utils.GetInstanceAndHost(event)
	if err != nil {
		return errors.WithStack(err)
	}

	if noOp, ok := utils.GetFieldsIfExist(event.Data, "processData", "containerNoOpEvent"); ok {
		instance.ProcessData.ContainerNoOpEvent = utils.InterfaceToBool(noOp)
	}

	if ok, err := compute.IsInstanceRemoved(instance, h.dockerClient); ok {
		return reply(map[string]interface{}{}, event, cli)
	} else if err != nil {
		return errors.WithStack(err)
	}

	if err := compute.DoInstanceRemove(instance, h.dockerClient); err != nil {
		return errors.WithStack(err)
	}
	return h.reply(event, cli)
}

func (h *ComputeHandler) reply(event *revents.Event, cli *client.RancherClient) error {
	resp, err := utils.GetResponseData(event, h.dockerClient)
	if err != nil {
		return errors.WithStack(err)
	}
	return reply(resp, event, cli)
}
