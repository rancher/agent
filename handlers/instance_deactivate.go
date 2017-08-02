package handlers

import (
	"github.com/pkg/errors"
	"github.com/rancher/agent/runtime"
	"github.com/rancher/agent/utils"
	revents "github.com/rancher/event-subscriber/events"
	v2 "github.com/rancher/go-rancher/v2"
)

func (h *ComputeHandler) InstanceDeactivate(event *revents.Event, cli *v2.RancherClient) error {
	request, err := utils.GetDeploymentSyncRequest(event)
	if err != nil {
		return errors.Wrap(err, "failed to marshall deploymentSyncRequest from event")
	}
	if len(request.Containers) == 0 {
		return errors.New("the number of instances for deploymentSyncRequest is zero")
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
		if stopped, err := runtime.IsContainerStopped(request.Containers[0], h.dockerClient); err != nil {
			return errors.Wrap(err, "failed to check whether instance is activated")
		} else if !stopped {
			timeout, ok := utils.GetFieldsIfExist(event.Data, "processData", "timeout")
			if !ok {
				timeout = 10
			}
			switch timeout.(type) {
			case float64:
				timeout = int(timeout.(float64))
			}
			err = runtime.ContainerStop(request.Containers[0], request.Volumes, h.dockerClient, timeout.(int))
			if err != nil {
				return errors.Wrap(err, "failed to deactivate instance")
			}
		}
	}

	response, err := constructDeploymentSyncReply(request.Containers[0], h.dockerClient, networkKind, nil)
	if err != nil {
		return errors.Wrap(err, "failed to construct deploymentSyncResponse")
	}
	data := map[string]interface{}{
		"deploymentSyncResponse": response,
	}
	return reply(data, event, cli)
}
