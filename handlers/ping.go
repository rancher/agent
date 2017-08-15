package handlers

import (
	"github.com/pkg/errors"
	"github.com/rancher/agent/ping"
	"github.com/rancher/agent/utils"
	revents "github.com/rancher/event-subscriber/events"
	v3 "github.com/rancher/go-rancher/v3"
)

func (h *PingHandler) Ping(event *revents.Event, cli *v3.RancherClient) error {
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
		"options":   resp.Options,
		"hashKey":   resp.HashKey,
	}
	return reply(data, event, cli)
}
