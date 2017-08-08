package handlers

import (
	"github.com/pkg/errors"
	"github.com/rancher/agent/runtime"
	"github.com/rancher/agent/utils"
	revents "github.com/rancher/event-subscriber/events"
	v3 "github.com/rancher/go-rancher/v3"
)

func (h *ComputeHandler) InstanceActivate(event *revents.Event, cli *v3.RancherClient) error {
	request, err := utils.GetDeploymentSyncRequest(event)
	if err != nil {
		return errors.Wrap(err, "failed to marshall deploymentSyncRequest from event")
	}
	if len(request.Containers) == 0 {
		return errors.New("the number of instances for deploymentSyncRequest is zero")
	}

	progress := utils.GetProgress(event, cli)
	idsMap := map[string]string{}
	for _, container := range request.Containers {
		if container.ExternalId != "" {
			idsMap[container.Id] = container.ExternalId
		}
	}

	networkKind := ""
	for _, network := range request.Networks {
		if request.Containers[0].PrimaryNetworkId == network.Id {
			networkKind = network.Kind
			break
		}
	}

	noop := false
	value, ok := utils.GetFieldsIfExist(event.Data, "processData", "containerNoOpEvent")
	if ok {
		noop = utils.InterfaceToBool(value)
	}

	if !noop {
		if started, err := runtime.IsContainerStarted(request.Containers[0], h.dockerClient); err == nil && !started {
			if err := runtime.ContainerStart(request.Containers[0], request.Volumes, networkKind, request.RegistryCredentials, progress, h.dockerClient, idsMap); err != nil {
				return errors.Wrap(err, "failed to activate instance")
			}
		} else if err != nil {
			return errors.Wrap(err, "failed to check whether instance is activated")
		}
	}

	response, err := constructDeploymentSyncReply(request.Containers[0], h.dockerClient, networkKind, progress)
	if err != nil {
		return errors.Wrap(err, "failed to construct deploymentSyncResponse")
	}
	data := map[string]interface{}{
		"deploymentSyncResponse": response,
	}
	return reply(data, event, cli)
}
