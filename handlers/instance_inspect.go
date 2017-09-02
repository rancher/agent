package handlers

import (
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"github.com/rancher/agent/runtime"
	"github.com/rancher/agent/utils"
	revents "github.com/rancher/event-subscriber/events"
	v3 "github.com/rancher/go-rancher/v3"
)

func (h *ComputeHandler) InstanceInspect(event *revents.Event, cli *v3.RancherClient) error {
	var inspect runtime.InstanceInspect
	err := utils.Unmarshalling(event.Data["instanceInspect"], &inspect)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshall instanceInspect")
	}
	inspectResp, err := runtime.ContainerInspect(inspect, h.dockerClient)
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return errors.Wrap(err, "failed to inspect instance")
	}
	if err == nil && strings.HasPrefix(inspectResp.Image, "sha256:") {
		if err := utils.ReplaceFriendlyImage(h.cache, h.dockerClient, &inspectResp); err != nil {
			return err
		}
	}
	if err == nil {
		logrus.Infof("rancher id [%v]: Container with docker id [%v] has been inspected", event.ResourceID, inspect.ID)
	}
	result := map[string]interface{}{event.ResourceType: inspectResp}
	return reply(result, event, cli)
}
