package handlers

import (
	"github.com/pkg/errors"
	"github.com/rancher/agent/runtime"
	"github.com/rancher/agent/utils"
	revents "github.com/rancher/event-subscriber/events"
	v3 "github.com/rancher/go-rancher/v3"
)

func (h *ComputeHandler) InstanceRemove(event *revents.Event, cli *v3.RancherClient) error {
	containerSpec, err := utils.GetContainerSpec(event)
	if err != nil {
		return errors.Wrap(err, "failed to marshall instance and host data")
	}

	if removed, err := runtime.IsContainerRemoved(containerSpec, h.dockerClient); err == nil && !removed {
		if err := runtime.ContainerRemove(containerSpec, h.dockerClient); err != nil {
			return errors.Wrap(err, "failed to remove instance")
		}
	} else if err != nil {
		return errors.Wrap(err, "failed to check whether instance is removed")
	}

	return reply(nil, event, cli)
}
