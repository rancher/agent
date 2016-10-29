//+build windows

package handlers

import (
	"github.com/docker/docker/api/types"
	"github.com/rancher/agent/utilities/constants"
	"github.com/rancher/agent/utilities/docker"
	"github.com/rancher/agent/utilities/utils"
	"golang.org/x/net/context"
	"gopkg.in/check.v1"
)

func (s *ComputeTestSuite) TestInstanceActivateWindowsImage(c *check.C) {
	deleteContainer("/c861f990-4472-4fa1-960f-65171b544c26")

	rawEvent := loadEvent("./test_events/instance_activate_windows", c)
	reply := testEvent(rawEvent, c)
	container, ok := utils.GetFieldsIfExist(reply.Data, "instanceHostMap", "instance", "+data", "dockerContainer")
	if !ok {
		c.Fatal("No ID found")
	}
	dockerClient := docker.GetClient(constants.DefaultVersion)
	inspect, err := dockerClient.ContainerInspect(context.Background(), container.(*types.Container).ID)
	if err != nil {
		c.Fatal("Inspect Err")
	}
	c.Check(inspect.Config.Image, check.Equals, "microsoft/iis:latest")
}

func (s *ComputeTestSuite) TestInstanceDeactivateWindowsImage(c *check.C) {
	deleteContainer("/c861f990-4472-4fa1-960f-65171b544c26")

	rawEvent := loadEvent("./test_events/instance_activate_windows", c)
	reply := testEvent(rawEvent, c)
	container, ok := utils.GetFieldsIfExist(reply.Data, "instanceHostMap", "instance", "+data", "dockerContainer")
	if !ok {
		c.Fatal("No ID found")
	}
	dockerClient := docker.GetClient(constants.DefaultVersion)
	inspect, err := dockerClient.ContainerInspect(context.Background(), container.(*types.Container).ID)
	if err != nil {
		c.Fatal("Inspect Err")
	}
	c.Check(inspect.Config.Image, check.Equals, "microsoft/iis:latest")

	rawEventDe := loadEvent("./test_events/instance_deactivate_windows", c)
	replyDe := testEvent(rawEventDe, c)
	container, ok = utils.GetFieldsIfExist(replyDe.Data, "instanceHostMap", "instance", "+data", "dockerContainer")
	if !ok {
		c.Fatal("No ID found")
	}
	inspect, err = dockerClient.ContainerInspect(context.Background(), container.(*types.Container).ID)
	if err != nil {
		c.Fatal("Inspect Err")
	}
	c.Check(inspect.State.Status, check.Equals, "exited")
}
