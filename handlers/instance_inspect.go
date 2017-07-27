package handlers

import (
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"github.com/rancher/agent/runtime"
	"github.com/rancher/agent/utils"
	revents "github.com/rancher/event-subscriber/events"
	v2 "github.com/rancher/go-rancher/v2"
)

func (h *ComputeHandler) InstanceInspect(event *revents.Event, cli *v2.RancherClient) error {
	var inspect runtime.InstanceInspect
	err := utils.Unmarshalling(event.Data["instanceInspect"], &inspect)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshall instanceInspect")
	}
	inspectResp, err := runtime.ContainerInspect(inspect, h.dockerClient)
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return errors.Wrap(err, "failed to inspect instance")
	}
	logrus.Infof("rancher id [%v]: Container with docker id [%v] has been inspected", event.ResourceID, inspect.ID)
	inspectJSON, err := utils.StructToMap(inspectResp)
	if err != nil {
		return errors.Wrap(err, "failed to marshall response data")
	}
	result := map[string]interface{}{event.ResourceType: inspectJSON}
	return reply(result, event, cli)
}
