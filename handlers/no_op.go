package handlers

import (
	"github.com/Sirupsen/logrus"
	revents "github.com/rancher/go-machine-service/events"
	"github.com/rancher/go-rancher/client"
)

func NoOpHandler(event *revents.Event, cli *client.RancherClient) error {
	logrus.Infof("Received and ignoring event: Name: %s, Event Id: %s, Resource Id: %s", event.Name, event.ID, event.ResourceID)
	return reply(nil, event, cli)
}
