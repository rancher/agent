package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	revents "github.com/rancher/go-machine-service/events"
	"github.com/rancher/go-rancher/client"
	"gopkg.in/check.v1"
	"github.com/Sirupsen/logrus"
	"strings"
)

func loadEvent(eventFile string, c *check.C) []byte {
	file, err := ioutil.ReadFile(eventFile)
	if err != nil {
		c.Fatalf("Error reading event %v", err)
	}
	return file

}


func getInstance(event map[string]interface{}, c *check.C) map[string]interface{} {
	data := event["data"].(map[string]interface{})
	ihm := data["instanceHostMap"].(map[string]interface{})
	instance := ihm["instance"].(map[string]interface{})
	return instance
}


func unmarshalEvent(rawEvent []byte, c *check.C) map[string]interface{} {
	event := map[string]interface{}{}
	err := json.Unmarshal(rawEvent, &event)
	if err != nil {
		c.Fatalf("Error unmarshalling event %v", err)
	}
	return event
}

func marshalEvent(event interface{}, c *check.C) []byte {
	b, err := json.Marshal(event)
	if err != nil {
		c.Fatalf("Error marshalling event %v", err)
	}
	return b
}

func testEvent(rawEvent []byte, c *check.C) *client.Publish {
	apiClient, mockPublish := newTestClient()
	workers := make(chan *revents.Worker, 1)
	worker := &revents.Worker{}
	worker.DoWork(rawEvent, GetHandlers(), apiClient, workers)
	return mockPublish.publishedResponse
}

func newTestClient() (*client.RancherClient, *mockPublishOperations) {
	mock := &mockPublishOperations{}
	return &client.RancherClient{
		Publish: mock,
	}, mock
}

/*
type PublishOperations interface {
	List(opts *ListOpts) (*PublishCollection, error)
	Create(opts *Publish) (*Publish, error)
	Update(existing *Publish, updates interface{}) (*Publish, error)
	ById(id string) (*Publish, error)
	Delete(container *Publish) error
}
*/
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
