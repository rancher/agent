package ping

import (
	"github.com/docker/engine-api/client"
	"github.com/pkg/errors"
	"github.com/rancher/agent/core/hostInfo"
	"github.com/rancher/agent/model"
	"github.com/rancher/agent/utilities/config"
	revents "github.com/rancher/event-subscriber/events"
)

func DoPingAction(event *revents.Event, resp *model.PingResponse, dockerClient *client.Client, collectors []hostInfo.Collector, systemImages map[string]string) error {
	if !config.DockerEnable() {
		return nil
	}
	if err := addResource(event, resp, dockerClient, collectors); err != nil {
		return errors.WithStack(err)
	}
	if err := addInstance(event, resp, dockerClient, systemImages); err != nil {
		return errors.WithStack(err)
	}
	return nil
}
