package handlers

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	revents "github.com/rancher/go-machine-service/events"
	"github.com/rancher/go-rancher/client"
	"gopkg.in/check.v1"
	"strings"
)

func GetHandlers() map[string]revents.EventHandler {
	return map[string]revents.EventHandler{
		"compute.instance.activate":  InstanceActivate,
		"compute.intance.deactivate": NoOpHandler,
		"compute.force.stop":         NoOpHandler,
		"compute.instance.inspect":   NoOpHandler,
		"compute.instance.pull":      NoOpHandler,
		"compute.instance.remove":    NoOpHandler,
		"storage.image.activate":     NoOpHandler,
		"storage.volume.activate":    NoOpHandler,
		"storage.volume.deactivate":  NoOpHandler,
		"storage.volume.remove":      NoOpHandler,
		"delegate.request":           NoOpHandler,
		"ping":                       NoOpHandler,
	}
}

func reply(replyData map[string]interface{}, event *revents.Event, cli *client.RancherClient) error {
	if replyData == nil {
		replyData = make(map[string]interface{})
	}

	reply := &client.Publish{
		ResourceId:   event.ResourceID,
		PreviousIds:  []string{event.ID},
		ResourceType: event.ResourceType,
		Name:         event.ReplyTo,
		Data:         replyData,
	}

	logrus.Infof("Reply: %+v", reply)
	err := publishReply(reply, cli)
	if err != nil {
		return fmt.Errorf("Error sending reply %v: %v", event.ID, err)
	}
	return nil
}

func publishReply(reply *client.Publish, apiClient *client.RancherClient) error {
	_, err := apiClient.Publish.Create(reply)
	return err
}
