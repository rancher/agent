package framework

import (
	"encoding/json"
	"fmt"
	"github.com/rancher/agent/handlers"
	revents "github.com/rancher/event-subscriber/events"
	"github.com/rancher/event-subscriber/locks"
	"github.com/rancher/go-rancher/v2"
	"github.com/rancher/log"
)

func testEvent(rawEvent []byte) *client.Publish {
	apiClient, mockPublish := newTestClient()
	workers := make(chan *Worker, 1)
	worker := &Worker{}
	h, _ := handlers.GetHandlers()
	worker.DoWork(rawEvent, h, apiClient, workers)
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
	return nil, fmt.Errorf("mock not implemented")
}

func (m *mockPublishOperations) Update(existing *client.Publish, updates interface{}) (*client.Publish, error) {
	return nil, fmt.Errorf("mock not implemented")
}

func (m *mockPublishOperations) ById(id string) (*client.Publish, error) { // golint_ignore
	return nil, fmt.Errorf("mock not implemented")
}

func (m *mockPublishOperations) Delete(existing *client.Publish) error {
	return fmt.Errorf("mock not implemented")
}

type Worker struct {
}

func (w *Worker) DoWork(rawEvent []byte, eventHandlers map[string]revents.EventHandler, apiClient *client.RancherClient,
	workers chan *Worker) {
	defer func() { workers <- w }()

	event := &revents.Event{}
	err := json.Unmarshal(rawEvent, &event)
	if err != nil {
		log.Errorf("Error unmarshalling event error=%v", err)
		return
	}

	if event.Name != "ping" {
		log.Debugf("Processing event: %v", string(rawEvent[:]))
	}

	unlocker := locks.Lock(event.ResourceID)
	if unlocker == nil {
		log.Debugf("Resource (resourceId: %v) locked. Dropping event", event.ResourceID)
		return
	}
	defer unlocker.Unlock()

	if fn, ok := eventHandlers[event.Name]; ok {
		err = fn(event, apiClient)
		if err != nil {
			log.Errorf("Error processing event(eventName=%v, eventId=%v, resourceId=%v) error=%v", event.Name, event.ID, event.ResourceID, err)

			reply := &client.Publish{
				Name:                 event.ReplyTo,
				PreviousIds:          []string{event.ID},
				Transitioning:        "error",
				TransitioningMessage: err.Error(),
			}
			_, err := apiClient.Publish.Create(reply)
			if err != nil {
				log.Errorf("Error sending error-reply: %v", err)
			}
		}
	} else {
		log.Warn("No event handler registered for event (eventName=%v)", event.Name)
	}
}
