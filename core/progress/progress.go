package progress

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	revents "github.com/rancher/event-subscriber/events"
	"github.com/rancher/go-rancher/v2"
)

type Progress struct {
	Request *revents.Event
	Client  *client.RancherClient
}

func (p *Progress) Update(msg string, types string, data map[string]interface{}) {
	resp := &client.Publish{
		ResourceId:            p.Request.ResourceID,
		PreviousIds:           []string{p.Request.ID},
		ResourceType:          p.Request.ResourceType,
		Name:                  p.Request.ReplyTo,
		Data:                  data,
		Transitioning:         types,
		TransitioningMessage:  msg,
		TransitioningProgress: 0,
	}
	transition := fmt.Sprintf("%s: %s", resp.Transitioning, resp.TransitioningMessage)
	empty := "empty"
	logrus.Infof("Reply: %v, %v, %v:%v, data: %v %v", p.Request.ID, p.Request.ReplyTo, resp.ResourceId, resp.ResourceType, empty, transition)
	err := publishReply(resp, p.Client)
	if err != nil {
		logrus.Error(err)
	}
}

func (p *Progress) UpdateWithParent(msg string, types string, data map[string]interface{}, parent *revents.Event) {
	resp := &client.Publish{
		ResourceId:            parent.ResourceID,
		PreviousIds:           []string{parent.ID},
		ResourceType:          parent.ResourceType,
		Name:                  parent.ReplyTo,
		Data:                  data,
		Transitioning:         types,
		TransitioningMessage:  msg,
		TransitioningProgress: 0,
	}
	transition := fmt.Sprintf("%s: %s", resp.Transitioning, resp.TransitioningMessage)
	empty := "empty"
	logrus.Infof("Reply: %v, %v, %v:%v, data: %v %v", p.Request.ID, p.Request.ReplyTo, resp.ResourceId, resp.ResourceType, empty, transition)
	err := publishReply(resp, p.Client)
	if err != nil {
		logrus.Error(err)
	}
}

func publishReply(reply *client.Publish, apiClient *client.RancherClient) error {
	_, err := apiClient.Publish.Create(reply)
	return err
}
