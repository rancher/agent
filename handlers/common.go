package handlers

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	goUUID "github.com/nu7hatch/gouuid"
	revents "github.com/rancher/go-machine-service/events"
	"github.com/rancher/go-rancher/client"
	"time"
)

func GetHandlers() map[string]revents.EventHandler {
	return map[string]revents.EventHandler{
		"compute.instance.activate":   InstanceActivate,
		"compute.instance.deactivate": InstanceDeactivate,
		"compute.instance.force.stop": InstanceForceStop,
		"compute.instance.inspect":    InstanceInspect,
		"compute.instance.pull":       InstancePull,
		"compute.instance.remove":     InstanceRemove,
		"storage.image.activate":      ImageActivate,
		"storage.volume.activate":     VolumeActivate,
		"storage.volume.deactivate":   VolumeDeactivate,
		"storage.volume.remove":       VolumeRemove,
		"delegate.request":            DelegateRequest,
		"ping":                        Ping,
		"config.update":               ConfigUpdate,
	}
}

func reply(replyData map[string]interface{}, event *revents.Event, cli *client.RancherClient) error {
	if replyData == nil {
		replyData = make(map[string]interface{})
	}

	reply := &client.Publish{
		ResourceId:    event.ResourceID,
		PreviousIds:   []string{event.ID},
		ResourceType:  event.ResourceType,
		Name:          event.ReplyTo,
		Data:          replyData,
		Time:          time.Now().UnixNano() / int64(time.Millisecond),
		Resource:      client.Resource{Id: getUUID()},
		PreviousNames: []string{event.Name},
	}

	logrus.Infof("Reply: %+v", reply)
	err := publishReply(reply, cli)
	if err != nil {
		return fmt.Errorf("Error sending reply %v: %v", event.ID, err)
	}
	return nil
}

func replyWithParent(replyData map[string]interface{}, event *revents.Event, parent *revents.Event, cli *client.RancherClient) error {
	child := map[string]interface{}{
		"resourceId":    event.ResourceID,
		"previousIds":   []string{event.ID},
		"resourceType":  event.ResourceType,
		"name":          event.ReplyTo,
		"data":          replyData,
		"id":            getUUID(),
		"time":          time.Now().UnixNano() / int64(time.Millisecond),
		"previousNames": []string{event.Name},
	}
	reply := &client.Publish{
		ResourceId:    parent.ResourceID,
		PreviousIds:   []string{parent.ID},
		ResourceType:  parent.ResourceType,
		Name:          parent.ReplyTo,
		Data:          child,
		Time:          time.Now().UnixNano() / int64(time.Millisecond),
		Resource:      client.Resource{Id: getUUID()},
		PreviousNames: []string{parent.Name},
	}
	if parent.ReplyTo == "" {
		return nil
	}
	logrus.Infof("Reply: %+v", reply)
	err := publishReply(reply, cli)
	if err != nil {
		return fmt.Errorf("Error sending reply %v: %v", event.ID, err)
	}
	return nil
}

func getUUID() string {
	uuid := ""
	newUUID, err1 := goUUID.NewV4()
	if err1 != nil {
		logrus.Error(err1)
	} else {
		uuid = newUUID.String()
	}
	return uuid
}

func publishReply(reply *client.Publish, apiClient *client.RancherClient) error {
	resp, err := apiClient.Publish.Create(reply)
	logrus.Infof("response data %+v", resp)
	if err != nil {
		logrus.Error(err)
	}
	return err
}
