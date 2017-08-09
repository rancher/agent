package progress

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	revents "github.com/rancher/event-subscriber/events"
	v3 "github.com/rancher/go-rancher/v3"
)

type Progress struct {
	Request *revents.Event
	Client  *v3.RancherClient
}

func (p *Progress) Update(msg string, types string, data map[string]interface{}) {
	resp := &v3.Publish{
		ResourceId:           p.Request.ResourceID,
		PreviousIds:          []string{p.Request.ID},
		ResourceType:         p.Request.ResourceType,
		Name:                 p.Request.ReplyTo,
		Data:                 data,
		Transitioning:        types,
		TransitioningMessage: msg,
	}
	transition := fmt.Sprintf("%s: %s", resp.Transitioning, resp.TransitioningMessage)
	logrus.Debugf("Reply: %v, %v, %v:%v, transitioning: %v", p.Request.ID, p.Request.Name, resp.ResourceId, resp.ResourceType, transition)
	err := publishReply(resp, p.Client)
	if err != nil {
		logrus.Error(err)
	}
}

func publishReply(reply *v3.Publish, apiClient *v3.RancherClient) error {
	_, err := apiClient.Publish.Create(reply)
	return err
}
