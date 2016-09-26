package framework

import (
	"fmt"
	"github.com/rancher/agent/handlers"
	revents "github.com/rancher/event-subscriber/events"
	"github.com/rancher/go-rancher/v2"
)

func testEvent(rawEvent []byte) *client.Publish {
	apiClient, mockPublish := newTestClient()
	workers := make(chan *revents.Worker, 1)
	worker := &revents.Worker{}
	worker.DoWork(rawEvent, handlers.GetHandlers(), apiClient, workers)
	return mockPublish.publishedResponse
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
