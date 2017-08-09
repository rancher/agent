package handlers

import (
	"encoding/json"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/rancher/agent/utils"
	"github.com/rancher/event-subscriber/events"
	"golang.org/x/net/context"
	"gopkg.in/check.v1"
)

func (s *EventTestSuite) TestInstanceInspectByName(c *check.C) {
	deleteContainer("/inspect_test")

	cli := utils.GetRuntimeClient("docker", utils.DefaultVersion)
	config := &container.Config{
		Image:     "ibuildthecloud/helloworld:latest",
		OpenStdin: true,
	}
	hostConfig := &container.HostConfig{}
	resp, err := cli.ContainerCreate(context.Background(), config, hostConfig, nil, "inspect_test")
	if err != nil {
		c.Fatal(err)
	}
	err = cli.ContainerStart(context.Background(), resp.ID, types.ContainerStartOptions{})
	if err != nil {
		c.Fatal(err)
	}

	rawEvent := loadEvent("./test_events/instance_inspect", c)
	reply := testEvent(rawEvent, c)

	inspect, err := cli.ContainerInspect(context.Background(), resp.ID)
	if err != nil {
		c.Fatal(err)
	}

	c.Assert(reply.Transitioning != "error", check.Equals, true)
	c.Assert(reply.Data["instanceInspect"].(types.ContainerJSON).ID, check.Equals, inspect.ID)
	c.Assert(reply.Data["instanceInspect"].(types.ContainerJSON).Image, check.Equals, inspect.Image)
}

func (s *EventTestSuite) TestInstanceInspectById(c *check.C) {
	deleteContainer("/inspect_test")

	cli := utils.GetRuntimeClient("docker", utils.DefaultVersion)
	config := &container.Config{
		Image:     "ibuildthecloud/helloworld:latest",
		OpenStdin: true,
	}
	hostConfig := &container.HostConfig{}
	resp, err := cli.ContainerCreate(context.Background(), config, hostConfig, nil, "inspect_test")
	if err != nil {
		c.Fatal(err)
	}
	err = cli.ContainerStart(context.Background(), resp.ID, types.ContainerStartOptions{})
	if err != nil {
		c.Fatal(err)
	}

	rawEvent := loadEvent("./test_events/instance_inspect", c)
	var event events.Event
	if err := json.Unmarshal(rawEvent, &event); err != nil {
		c.Fatal(err)
	}
	event.Data["instanceInspect"] = map[string]interface{}{
		"id":   resp.ID,
		"kind": "docker",
	}
	reply := testEvent(rawEvent, c)

	inspect, err := cli.ContainerInspect(context.Background(), resp.ID)
	if err != nil {
		c.Fatal(err)
	}

	c.Assert(reply.Transitioning != "error", check.Equals, true)
	c.Assert(reply.Data["instanceInspect"].(types.ContainerJSON).ID, check.Equals, inspect.ID)
	c.Assert(reply.Data["instanceInspect"].(types.ContainerJSON).Image, check.Equals, inspect.Image)
}
