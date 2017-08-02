package handlers

import (
	"github.com/pkg/errors"
	"github.com/rancher/agent/ping"
	"github.com/rancher/agent/utils"
	revents "github.com/rancher/event-subscriber/events"
	v2 "github.com/rancher/go-rancher/v2"
)

func (h *PingHandler) Ping(event *revents.Event, cli *v2.RancherClient) error {
	if event.ReplyTo == "" {
		return nil
	}
	resp := ping.Response{
		Resources: []ping.Resource{},
	}
	if utils.DoPing() {
		if err := ping.DoPingAction(event, &resp, h.dockerClient, h.collectors); err != nil {
			return errors.Wrap(err, "failed to do ping action")
		}
	}
	data := map[string]interface{}{
		"resources": resp.Resources,
	}
	return reply(data, event, cli)
}
