package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/api/types"
	"github.com/rancher/agent/utils"
	revents "github.com/rancher/event-subscriber/events"
	"github.com/rancher/event-subscriber/locks"
	"github.com/rancher/go-rancher/v3"
	"golang.org/x/net/context"
	"gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) {
	check.TestingT(t)
}

type EventTestSuite struct {
}

var _ = check.Suite(&EventTestSuite{})

func (s *EventTestSuite) SetUpSuite(c *check.C) {
}

func getDeploymentSyncRequest(path string, request interface{}, c *check.C) revents.Event {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		c.Fatalf("Error reading event %v", err)
	}
	var event revents.Event
	if err := json.Unmarshal(file, &event); err != nil {
		c.Fatal(err)
	}
	if err := utils.Unmarshalling(event.Data["deploymentSyncRequest"], request); err != nil {
		c.Fatal(err)
	}
	return event
}

func getDockerInspect(reply *client.Publish, c *check.C) types.ContainerJSON {
	response := reply.Data["deploymentSyncResponse"].(client.DeploymentSyncResponse)
	c.Assert(response.InstanceStatus, check.HasLen, 1)
	var inspect types.ContainerJSON
	if err := utils.Unmarshalling(response.InstanceStatus[0].DockerInspect, &inspect); err != nil {
		c.Fatal(err)
	}
	return inspect
}

func loadEvent(eventFile string, c *check.C) []byte {
	file, err := ioutil.ReadFile(eventFile)
	if err != nil {
		c.Fatalf("Error reading event %v", err)
	}
	return file
}

func findContainer(uuid string) types.Container {
	dockerClient := utils.GetRuntimeClient("docker", utils.DefaultVersion)
	containerList, _ := dockerClient.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	for _, c := range containerList {
		found := false
		labels := c.Labels
		if labels["io.rancher.container.uuid"] == uuid {
			found = true
		}

		for _, cname := range c.Names {
			if uuid == cname {
				found = true
				break
			}
		}
		if found {
			return c
		}
	}
	return types.Container{}
}

func deleteContainer(uuid string) {
	dockerClient := utils.GetRuntimeClient("docker", utils.DefaultVersion)
	containerList, _ := dockerClient.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	for _, c := range containerList {
		found := false
		labels := c.Labels
		if labels["io.rancher.container.uuid"] == uuid {
			found = true
		}

		for _, cname := range c.Names {
			if uuid == cname {
				found = true
				break
			}
		}
		if found {
			dockerClient.ContainerKill(context.Background(), c.ID, "KILL")
			for i := 0; i < 10; i++ {
				if inspect, err := dockerClient.ContainerInspect(context.Background(), c.ID); err == nil && inspect.State.Pid == 0 {
					break
				}
				time.Sleep(time.Duration(500) * time.Millisecond)
			}
			dockerClient.ContainerRemove(context.Background(), c.ID, types.ContainerRemoveOptions{})
		}
	}
}

func checkStringInArray(array []string, item string) bool {
	for _, str := range array {
		if str == item {
			return true
		}
	}
	return false
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

func unmarshalEventAndInstanceFields(rawEvent []byte, c *check.C) (map[string]interface{}, map[string]interface{},
	map[string]interface{}) {
	event := unmarshalEvent(rawEvent, c)
	instance := event["data"].(map[string]interface{})["instance"].(map[string]interface{})
	fields := instance["data"].(map[string]interface{})["fields"].(map[string]interface{})
	return event, instance, fields
}

func testEvent(rawEvent []byte, c *check.C) *client.Publish {
	apiClient, mockPublish := newTestClient()
	workers := make(chan *Worker, 1)
	worker := &Worker{}
	handlers, _ := GetHandlers()
	worker.DoWork(rawEvent, handlers, apiClient, workers)
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
