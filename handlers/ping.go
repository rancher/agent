package handlers

import (
	revents "github.com/rancher/go-machine-service/events"
	"github.com/rancher/go-rancher/client"
	//"errors"
	//"github.com/nu7hatch/gouuid"
	//"github.com/rancher/agent/handlers/utils"
	"errors"
	"github.com/Sirupsen/logrus"
	"github.com/rancher/agent/handlers/utils"
)

func Ping(event *revents.Event, cli *client.RancherClient) error {
	logrus.Infof("Received event: Name: %s, Event Id: %s, Resource Id: %s", event.Name, event.ID, event.ResourceID)
	if event.Name != "ping" || event.ReplyTo == "" {
		return errors.New("event name is invalid")
	}
	resp := utils.ReplyData(event)
	if utils.DoPing() {
		utils.DoPingAction(event, resp)
	}
	logrus.Infof("response data %v", resp)
	return reply(resp.Data, event, cli)
}
