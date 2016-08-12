package handlers

import (
	"github.com/Sirupsen/logrus"
	"github.com/rancher/agent/core/ping"
	"github.com/rancher/agent/utilities/config"
	revents "github.com/rancher/event-subscriber/events"
	"github.com/rancher/go-rancher/client"
)

func Ping(event *revents.Event, cli *client.RancherClient) error {
	logrus.Infof("Received event: Name: %s, Event Id: %s, Resource Id: %s", event.Name, event.ID, event.ResourceID)
	if event.Name != "ping" || event.ReplyTo == "" {
		return nil
	}
	resp := ping.ReplyData(event)
	if config.DoPing() {
		ping.DoPingAction(event, resp)
	}
	return reply(resp.Data, event, cli)
}
