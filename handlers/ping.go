package handlers

import (
	"github.com/Sirupsen/logrus"
	"github.com/rancher/agent/handlers/utils"
	revents "github.com/rancher/go-machine-service/events"
	"github.com/rancher/go-rancher/client"
)

func Ping(event *revents.Event, cli *client.RancherClient) error {
	logrus.Infof("Received event: Name: %s, Event Id: %s, Resource Id: %s", event.Name, event.ID, event.ResourceID)
	if event.Name != "ping" || event.ReplyTo == "" {
		return nil
	}
	resp := utils.ReplyData(event)
	if utils.DoPing() {
		utils.DoPingAction(event, resp)
	}
	return reply(resp.Data, event, cli)
}
