package handlers

import (
	"strings"

	"github.com/Sirupsen/logrus"
	engineCli "github.com/docker/docker/client"
	"github.com/mitchellh/mapstructure"
	cache "github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	"github.com/rancher/agent/core/compute"
	"github.com/rancher/agent/core/marshaller"
	"github.com/rancher/agent/model"
	"github.com/rancher/agent/utilities/constants"
	"github.com/rancher/agent/utilities/utils"
	revents "github.com/rancher/event-subscriber/events"
	"github.com/rancher/go-rancher/v2"
)

type ComputeHandler struct {
	dockerClient            *engineCli.Client
	dockerClientWithTimeout *engineCli.Client
	infoData                model.InfoData
	memCache                *cache.Cache
}

func (h *ComputeHandler) InstanceActivate(event *revents.Event, cli *client.RancherClient) error {
	instance, host, err := utils.GetInstanceAndHost(event)
	if err != nil {
		return errors.Wrap(err, constants.InstanceActivateError+"failed to marshall instance and host data")
	}

	progress := utils.GetProgress(event, cli)

	if noOp, ok := utils.GetFieldsIfExist(event.Data, "processData", "containerNoOpEvent"); ok {
		instance.ProcessData.ContainerNoOpEvent = utils.InterfaceToBool(noOp)
	}

	if ok, err := compute.IsInstanceActive(instance, host, h.dockerClientWithTimeout); ok {
		return instanceHostMapReply(event, cli, h.dockerClientWithTimeout, h.memCache)
	} else if err != nil {
		return errors.Wrap(err, constants.InstanceActivateError+"failed to check whether instance is activated")
	}

	if err := compute.DoInstanceActivate(instance, host, progress, h.dockerClient, h.infoData); err != nil {
		return errors.Wrap(err, constants.InstanceActivateError+"failed to activate instance")
	}
	return instanceHostMapReply(event, cli, h.dockerClientWithTimeout, h.memCache)
}

func (h *ComputeHandler) InstanceDeactivate(event *revents.Event, cli *client.RancherClient) error {
	instance, _, err := utils.GetInstanceAndHost(event)
	if err != nil {
		return errors.Wrap(err, constants.InstanceDeactivateError+"failed to marshall instance and host data")
	}

	if noOp, ok := utils.GetFieldsIfExist(event.Data, "processData", "containerNoOpEvent"); ok {
		instance.ProcessData.ContainerNoOpEvent = utils.InterfaceToBool(noOp)
	}

	if ok, err := compute.IsInstanceInactive(instance, h.dockerClient); err != nil {
		return errors.Wrap(err, constants.InstanceDeactivateError+"failed to check whether instance is activated")
	} else if ok {
		return instanceHostMapReply(event, cli, h.dockerClient, nil)
	}

	timeout, ok := utils.GetFieldsIfExist(event.Data, "processData", "timeout")
	if !ok {
		timeout = 0
	}
	switch timeout.(type) {
	case float64:
		timeout = int(timeout.(float64))
	}
	err = compute.DoInstanceDeactivate(instance, h.dockerClient, timeout.(int))
	if err != nil {
		return errors.Wrap(err, constants.InstanceDeactivateError+"failed to deactivate instance")
	}
	return instanceHostMapReply(event, cli, h.dockerClient, nil)
}

func (h *ComputeHandler) InstanceForceStop(event *revents.Event, cli *client.RancherClient) error {
	var request model.InstanceForceStop
	err := mapstructure.Decode(event.Data["instanceForceStop"], &request)
	if err != nil {
		return errors.Wrap(err, constants.InstanceForceStopError+"failed to marshall incoming request")
	}
	err = compute.DoInstanceForceStop(request, h.dockerClient)
	if err != nil {
		return errors.Wrap(err, constants.InstanceForceStopError+"failed to force stop container")
	}
	logrus.Infof("rancher id [%v]: Container with docker id [%v] has been deactivated", event.ResourceID, request.ID)
	return nil
}

func (h *ComputeHandler) InstanceInspect(event *revents.Event, cli *client.RancherClient) error {
	var inspect model.InstanceInspect
	if err := mapstructure.Decode(event.Data["instanceInspect"], &inspect); err != nil {
		return errors.Wrap(err, constants.InstanceInspectError+"failed to marshall incoming request")
	}
	inspectResp, err := compute.DoInstanceInspect(inspect, h.dockerClientWithTimeout)
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return errors.Wrap(err, constants.InstanceInspectError+"failed to inspect instance")
	}
	logrus.Infof("rancher id [%v]: Container with docker id [%v] has been inspected", event.ResourceID, inspect.ID)
	inspectJSON, err := marshaller.StructToMap(inspectResp)
	if err != nil {
		return errors.Wrap(err, constants.InstanceInspectError+"failed to marshall response data")
	}
	result := map[string]interface{}{event.ResourceType: inspectJSON}
	return reply(result, event, cli)
}

func (h *ComputeHandler) InstancePull(event *revents.Event, cli *client.RancherClient) error {
	progress := utils.GetProgress(event, cli)
	var instancePull model.InstancePull
	err := mapstructure.Decode(event.Data["instancePull"], &instancePull)
	if err != nil {
		return errors.Wrap(err, constants.InstancePullError+"failed to marshall incoming request")
	}
	imageUUID := instancePull.Image.Data.DockerImage.FullName
	if instancePull.Image.Data.DockerImage.Server != "" {
		imageUUID = instancePull.Image.Data.DockerImage.Server + "/" + imageUUID
	}
	imageParams := model.ImageParams{
		Image:     instancePull.Image,
		Mode:      instancePull.Mode,
		Complete:  instancePull.Complete,
		Tag:       instancePull.Tag,
		ImageUUID: imageUUID,
	}

	inspect, pullErr := compute.DoInstancePull(imageParams, progress, h.dockerClient)
	if pullErr != nil {
		return errors.Wrap(pullErr, constants.InstancePullError+"failed to pull instance")
	}
	logrus.Infof("rancher id [%v]: Image with docker id [%v] has been pulled", event.ResourceID, inspect.ID)
	return instancePullReply(event, cli, h.dockerClient)
}

func (h *ComputeHandler) InstanceRemove(event *revents.Event, cli *client.RancherClient) error {
	instance, _, err := utils.GetInstanceAndHost(event)
	if err != nil {
		return errors.Wrap(err, constants.InstanceRemoveError+"failed to marshall instance and host data")
	}

	if noOp, ok := utils.GetFieldsIfExist(event.Data, "processData", "containerNoOpEvent"); ok {
		instance.ProcessData.ContainerNoOpEvent = utils.InterfaceToBool(noOp)
	}

	if ok, err := compute.IsInstanceRemoved(instance, h.dockerClientWithTimeout); ok {
		return reply(map[string]interface{}{}, event, cli)
	} else if err != nil {
		return errors.Wrap(err, constants.InstanceRemoveError+"failed to check whether instance is removed")
	}

	if err := compute.DoInstanceRemove(instance, h.dockerClientWithTimeout); err != nil {
		return errors.Wrap(err, constants.InstanceRemoveError+"failed to remove instance")
	}
	return instanceHostMapReply(event, cli, h.dockerClientWithTimeout, nil)
}
