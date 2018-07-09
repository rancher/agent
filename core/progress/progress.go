package progress

import (
	"fmt"

	revents "github.com/rancher/event-subscriber/events"
	"github.com/rancher/go-rancher/v2"
	"github.com/rancher/log"
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
	log.Infof("Reply: %v, %v, %v:%v, transitioning: %v", p.Request.ID, p.Request.Name, resp.ResourceId, resp.ResourceType, transition)
	err := publishReply(resp, p.Client)
	if err != nil {
		log.Error(err)
	}
}

func (p *Progress) UpdateWithParent(msg string, types string, data map[string]interface{}, event, parent *revents.Event) {
	resp := &client.Publish{
		ResourceId:   parent.ResourceID,
		PreviousIds:  []string{parent.ID},
		ResourceType: parent.ResourceType,
		Name:         parent.ReplyTo,
		Data: map[string]interface{}{
			"resourceId":            event.ResourceID,
			"previousIds":           []string{event.ID},
			"resourceType":          event.ResourceType,
			"name":                  event.ReplyTo,
			"data":                  data,
			"transitioning":         types,
			"transitioningMessage":  msg,
			"transitioningProgress": 0,
		},
		Transitioning:         types,
		TransitioningMessage:  msg,
		TransitioningProgress: 0,
	}
	transition := fmt.Sprintf("%s: %s", resp.Transitioning, resp.TransitioningMessage)
	log.Infof("Reply: %v, %v, %v:%v, transitioning: %v", p.Request.ID, p.Request.ReplyTo, resp.ResourceId, resp.ResourceType, transition)
	err := publishReply(resp, p.Client)
	if err != nil {
		log.Error(err)
	}
}

func publishReply(reply *client.Publish, apiClient *client.RancherClient) error {
	_, err := apiClient.Publish.Create(reply)
	return err
}
