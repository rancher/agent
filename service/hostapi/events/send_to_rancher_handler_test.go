package events

import (
	"testing"
	"time"

	"github.com/docker/distribution/context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/patrickmn/go-cache"
	rclient "github.com/rancher/go-rancher/client"
)

func TestSendToRancherHandler(t *testing.T) {
	dockerClient := prep(t)

	injectedIP := "10.1.2.3"
	c, _ := createNetTestContainer(dockerClient, injectedIP)
	defer dockerClient.ContainerRemove(context.Background(), c.ID, types.ContainerRemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	})

	from := "foo/bar"
	status := "create"
	var eventTime int64 = 1426091566
	hostUUID := "host-123"
	event := &events.Message{ID: c.ID, From: from, Status: status, Time: eventTime}
	expectedEvent := &rclient.ContainerEvent{
		ExternalId:        c.ID,
		ExternalFrom:      from,
		ExternalStatus:    status,
		ExternalTimestamp: eventTime,
		ReportedHostUuid:  hostUUID,
	}
	rancher := mockRancherClient(expectedEvent, t)

	handler := &SendToRancherHandler{client: dockerClient, rancher: rancher, hostUUID: hostUUID, cache: cache.New(time.Hour*24, time.Hour*1)}

	if err := handler.Handle(event); err != nil {
		t.Fatal(err)
	}
}

func mockRancherClient(expectedEvent *rclient.ContainerEvent, t *testing.T) *rclient.RancherClient {
	apiClient := &rclient.RancherClient{
		ContainerEvent: &MockContainerEventOps{t: t, expectedEvent: expectedEvent},
	}

	return apiClient
}

type MockContainerEventOps struct {
	expectedEvent *rclient.ContainerEvent
	t             *testing.T
}

func (m *MockContainerEventOps) Create(event *rclient.ContainerEvent) (*rclient.ContainerEvent, error) {
	if event.ExternalId != m.expectedEvent.ExternalId ||
		event.ExternalFrom != m.expectedEvent.ExternalFrom ||
		event.ExternalTimestamp != m.expectedEvent.ExternalTimestamp ||
		event.ExternalStatus != m.expectedEvent.ExternalStatus ||
		event.ReportedHostUuid != m.expectedEvent.ReportedHostUuid ||
		event.DockerInspect == nil {
		m.t.Fatalf("Events don't match. Expected: %#v; Actual: %#v", m.expectedEvent, event)
	}
	return event, nil
}
func (m *MockContainerEventOps) List(opts *rclient.ListOpts) (*rclient.ContainerEventCollection, error) {
	return nil, nil
}
func (m *MockContainerEventOps) Update(existing *rclient.ContainerEvent,
	updates interface{}) (*rclient.ContainerEvent, error) {
	return nil, nil
}
func (m *MockContainerEventOps) ById(id string) (*rclient.ContainerEvent, error) {
	return nil, nil
}
func (m *MockContainerEventOps) Delete(container *rclient.ContainerEvent) error {
	return nil
}
func (m *MockContainerEventOps) ActionCreate(*rclient.ContainerEvent) (*rclient.ContainerEvent, error) {
	return nil, nil
}
func (m *MockContainerEventOps) ActionRemove(*rclient.ContainerEvent) (*rclient.ContainerEvent, error) {
	return nil, nil
}
