package progress

import (
	"github.com/Sirupsen/logrus"
	revents "github.com/rancher/go-machine-service/events"
	"github.com/rancher/go-rancher/client"
)

type Progress struct {
	Request *revents.Event
	Client  *client.RancherClient
}

func (p *Progress) Update(msg string) {
	resp := &client.Publish{
		ResourceId:            p.Request.ResourceID,
		PreviousIds:           []string{p.Request.ID},
		ResourceType:          p.Request.ResourceType,
		Name:                  p.Request.ReplyTo,
		Data:                  map[string]interface{}{},
		Transitioning:         "yes",
		TransitioningMessage:  msg,
		TransitioningProgress: 0,
	}

	err := publishReply(resp, p.Client)
	if err != nil {
		logrus.Error(err)
	}
}

func publishReply(reply *client.Publish, apiClient *client.RancherClient) error {
	_, err := apiClient.Publish.Create(reply)
	return err
}
