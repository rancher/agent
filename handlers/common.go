package handlers

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	revents "github.com/rancher/go-machine-service/events"
	"github.com/rancher/go-rancher/client"
)

func GetHandlers() map[string]revents.EventHandler {
	return map[string]revents.EventHandler{
		"compute.instance.activate":   InstanceActivate,
		"compute.instance.deactivate": InstanceDeactivate,
		"compute.force.stop":          InstanceForceStop,
		"compute.instance.inspect":    InstanceInspect,
		"compute.instance.pull":       InstancePull,
		"compute.instance.remove":     InstanceRemove,
		"storage.image.activate":      ImageActivate,
		"storage.volume.activate":     VolumeActivate,
		"storage.volume.deactivate":   VolumeDeactivate,
		"storage.volume.remove":       VolumeRemove,
		"delegate.request":            DelegateRequest,
		"ping":                        NoOpHandler,
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
