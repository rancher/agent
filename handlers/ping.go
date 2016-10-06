package handlers

import (
	engineCli "github.com/docker/engine-api/client"
	"github.com/pkg/errors"
	"github.com/rancher/agent/core/hostInfo"
	"github.com/rancher/agent/core/marshaller"
	"github.com/rancher/agent/core/ping"
	"github.com/rancher/agent/model"
	"github.com/rancher/agent/utilities/config"
	"github.com/rancher/agent/utilities/constants"
	revents "github.com/rancher/event-subscriber/events"
	"github.com/rancher/go-rancher/v2"
)

type PingHandler struct {
	dockerClient *engineCli.Client
	collectors   []hostInfo.Collector
	SystemImage  map[string]string
}

func (h *PingHandler) Ping(event *revents.Event, cli *client.RancherClient) error {
	if event.Name != "ping" || event.ReplyTo == "" {
		return nil
	}
	resp := model.PingResponse{
		Resources: []model.PingResource{},
	}
	if config.DoPing() {
		if err := ping.DoPingAction(event, &resp, h.dockerClient, h.collectors, h.SystemImage); err != nil {
			return errors.Wrap(err, constants.PingError+"failed to do ping action")
		}
	}
	data, err := marshaller.StructToMap(resp)
	if err != nil {
		return errors.Wrap(err, constants.PingError+"failed to marshall response data")
	}
	return reply(data, event, cli)
}
