package framework

import (
	"encoding/json"
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/agent/handlers"
	revents "github.com/rancher/event-subscriber/events"
	"github.com/rancher/event-subscriber/locks"
	"github.com/rancher/go-rancher/v2"
)

func testEvent(rawEvent []byte) *client.Publish {
	apiClient, mockPublish := newTestClient()
	workers := make(chan *Worker, 1)
	worker := &Worker{}
	worker.DoWork(rawEvent, handlers.GetHandlers(), apiClient, workers)
	return mockPublish.publishedResponse
}

func resourceIDLocker(event *revents.Event) locks.Locker {
	if event.ResourceID == "" {
		return locks.NopLocker()
	}
	key := fmt.Sprintf("%s:%s", event.ResourceType, event.ResourceID)
	return locks.KeyLocker(key)
}

func newTestClient() (*client.RancherClient, *mockPublishOperations) {
	mock := &mockPublishOperations{}
	return &client.RancherClient{
		Publish: mock,
	}, mock
}

type mockPublishOperations struct {
	publishedResponse *client.Publish
}

func (m *mockPublishOperations) Create(publish *client.Publish) (*client.Publish, error) {
	m.publishedResponse = publish
	return publish, nil
}

func (m *mockPublishOperations) List(publish *client.ListOpts) (*client.PublishCollection, error) {
	return nil, fmt.Errorf("Mock not implemented.")
}

func (m *mockPublishOperations) Update(existing *client.Publish, updates interface{}) (*client.Publish, error) {
	return nil, fmt.Errorf("Mock not implemented.")
}

func (m *mockPublishOperations) ById(id string) (*client.Publish, error) { // golint_ignore
	return nil, fmt.Errorf("Mock not implemented.")
}

func (m *mockPublishOperations) Delete(existing *client.Publish) error {
	return fmt.Errorf("Mock not implemented.")
}

type Worker struct {
}

func (w *Worker) DoWork(rawEvent []byte, eventHandlers map[string]revents.EventHandler, apiClient *client.RancherClient,
	workers chan *Worker) {
	defer func() { workers <- w }()

	event := &revents.Event{}
	err := json.Unmarshal(rawEvent, &event)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"err": err,
		}).Error("Error unmarshalling event")
		return
	}

	if event.Name != "ping" {
		logrus.WithFields(logrus.Fields{
			"event": string(rawEvent[:]),
		}).Debug("Processing event.")
	}

	unlocker := locks.Lock(event.ResourceID)
	if unlocker == nil {
		logrus.WithFields(logrus.Fields{
			"resourceId": event.ResourceID,
		}).Debug("Resource locked. Dropping event")
		return
	}
	defer unlocker.Unlock()

	if fn, ok := eventHandlers[event.Name]; ok {
		err = fn(event, apiClient)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"eventName":  event.Name,
				"eventId":    event.ID,
				"resourceId": event.ResourceID,
				"err":        err,
			}).Error("Error processing event")

			reply := &client.Publish{
				Name:                 event.ReplyTo,
				PreviousIds:          []string{event.ID},
				Transitioning:        "error",
				TransitioningMessage: err.Error(),
			}
			_, err := apiClient.Publish.Create(reply)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"err": err,
				}).Error("Error sending error-reply")
			}
		}
	} else {
		logrus.WithFields(logrus.Fields{
			"eventName": event.Name,
		}).Warn("No event handler registered for event")
	}
}
