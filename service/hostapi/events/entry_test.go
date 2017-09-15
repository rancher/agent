package events

import (
	"github.com/Sirupsen/logrus"
	"github.com/docker/distribution/context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"
	rclient "github.com/rancher/go-rancher/v3"
	"testing"
	"time"
)

func TestProcessDockerEvents(t *testing.T) {
	processor := NewDockerEventsProcessor(10)

	dockerClient, err := NewDockerClient()
	if err != nil {
		t.Fatal(err)
	}
	processor.getDockerClient = func() (*client.Client, error) {
		return dockerClient, nil
	}

	// Mock Handler
	handledEvents := make(chan *events.Message, 10)
	hFn := func(event *events.Message) error {
		handledEvents <- event
		return nil
	}
	handler := &testHandler{
		handlerFunc: hFn,
	}
	processor.getHandlers = func(dockerClient *client.Client,
		rancherClient *rclient.RancherClient) (map[string][]Handler, error) {
		return map[string][]Handler{"start": {handler}}, nil
	}

	// Create pre-existing containers before starting event listener
	preexistRunning, err := createNetTestContainer(dockerClient, "10.1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	logrus.Infof("%+v", preexistRunning)
	defer func() {
		if err := dockerClient.ContainerRemove(context.Background(), preexistRunning.ID, types.ContainerRemoveOptions{
			Force:         true,
			RemoveVolumes: true,
		}); err != nil {
			t.Fatal(err)
		}
	}()
	if err := dockerClient.ContainerStart(context.Background(), preexistRunning.ID, types.ContainerStartOptions{}); err != nil {
		t.Fatal(err)
	}
	preexistPaused, _ := createNetTestContainer(dockerClient, "10.1.2.3")
	defer func() {
		if err := dockerClient.ContainerUnpause(context.Background(), preexistPaused.ID); err != nil {
			t.Fatal(err)
		}
		if err := dockerClient.ContainerRemove(context.Background(), preexistPaused.ID, types.ContainerRemoveOptions{
			Force:         true,
			RemoveVolumes: true,
		}); err != nil {
			t.Fatal(err)
		}
	}()
	if err := dockerClient.ContainerStart(context.Background(), preexistPaused.ID, types.ContainerStartOptions{}); err != nil {
		t.Fatal(err)
	}
	dockerClient.ContainerPause(context.Background(), preexistPaused.ID)

	if err := processor.Process(); err != nil {
		t.Fatal(err)
	}

	waitingOnRunning := true
	waitingOnPaused := true
	for waitingOnRunning || waitingOnPaused {
		select {
		case e := <-handledEvents:
			if e.ID == preexistRunning.ID {
				waitingOnRunning = false
			}
			if e.ID == preexistPaused.ID {
				waitingOnPaused = false
			}
		case <-time.After(10 * time.Second):
			t.Fatalf("Never received event for preexisting container [%v]", preexistRunning.ID)
		}
	}
}

func TestGetHandlers(t *testing.T) {
	dockerClient := prep(t)
	handlers, err := getHandlersFn(dockerClient, nil)
	if err != nil {
		t.Fatal(err)
	}
	// RancherClient is not nil, so SendToRancherHandler should be configured
	handlers, err = getHandlersFn(dockerClient, &rclient.RancherClient{})
	if err != nil {
		t.Fatal(err)
	}
	if len(handlers) != 5 {
		t.Fatalf("Expected 5 configured hanlders: %v, %#v", len(handlers), handlers)
	}
}
