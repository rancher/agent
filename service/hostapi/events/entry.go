package events

import (
	"github.com/docker/distribution/context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/rancher/agent/service/hostapi/config"
	"github.com/rancher/agent/service/hostapi/util"
	rclient "github.com/rancher/go-rancher/client"
)

const (
	simulatedEvent = "-simulated-"
)

func NewDockerEventsProcessor(poolSize int) *DockerEventsProcessor {
	return &DockerEventsProcessor{
		poolSize:         poolSize,
		getDockerClient:  getDockerClientFn,
		getHandlers:      getHandlersFn,
		getRancherClient: util.GetRancherClient,
	}
}

type DockerEventsProcessor struct {
	poolSize         int
	getDockerClient  func() (*client.Client, error)
	getHandlers      func(*client.Client, *rclient.RancherClient) (map[string][]Handler, error)
	getRancherClient func() (*rclient.RancherClient, error)
}

func (de *DockerEventsProcessor) Process() error {
	dockerClient, err := de.getDockerClient()
	if err != nil {
		return err
	}

	rancherClient, err := de.getRancherClient()
	if err != nil {
		return err
	}

	handlers, err := de.getHandlers(dockerClient, rancherClient)
	if err != nil {
		return err
	}

	router, err := NewEventRouter(de.poolSize, de.poolSize, dockerClient, handlers)
	if err != nil {
		return err
	}
	router.Start()

	filter := filters.NewArgs()
	filter.Add("status", "paused")
	filter.Add("status", "running")
	listOpts := types.ContainerListOptions{
		All:    true,
		Filters: filter,
	}
	containers, err := dockerClient.ContainerList(context.Background(), listOpts)
	if err != nil {
		return err
	}

	for _, c := range containers {
		event := &events.Message{
			ID:     c.ID,
			Status: "start",
			From:   simulatedEvent,
		}
		router.listener <- event
	}
	return nil
}

func getDockerClientFn() (*client.Client, error) {
	return NewDockerClient()
}

func getHandlersFn(dockerClient *client.Client, rancherClient *rclient.RancherClient) (map[string][]Handler, error) {

	handlers := map[string][]Handler{}

	// Rancher Event Handler
	if rancherClient != nil {
		sendToRancherHandler := &SendToRancherHandler{
			client:   dockerClient,
			rancher:  rancherClient,
			hostUUID: getHostUUID(),
		}
		handlers["start"] = append(handlers["start"], sendToRancherHandler)
		handlers["stop"] = []Handler{sendToRancherHandler}
		handlers["die"] = []Handler{sendToRancherHandler}
		handlers["kill"] = []Handler{sendToRancherHandler}
		handlers["destroy"] = []Handler{sendToRancherHandler}
	}

	return handlers, nil
}

func getHostUUID() string {
	return config.Config.HostUUID
}
