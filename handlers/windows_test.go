package handlers

import (
	"gopkg.in/check.v1"
	"github.com/rancher/agent/utilities/utils"
	"github.com/rancher/agent/utilities/docker"
	"golang.org/x/net/context"
	"github.com/docker/engine-api/types"
	"runtime"
)

func (s *ComputeTestSuite) TestInstanceActivateWindowsImage(c *check.C) {
	if runtime.GOOS == "windows" {
		deleteContainer("/c861f990-4472-4fa1-960f-65171b544c29")

		rawEvent := loadEvent("./test_events/instance_activate_windows", c)
		reply := testEvent(rawEvent, c)
		container, ok := utils.GetFieldsIfExist(reply.Data, "instanceHostMap", "instance", "+data", "dockerContainer")
		if !ok {
			c.Fatal("No ID found")
		}
		inspect, err := docker.DefaultClient.ContainerInspect(context.Background(), container.(*types.Container).ID)
		if err != nil {
			c.Fatal("Inspect Err")
		}
		c.Check(inspect.Config.Image, check.Equals, "microsoft/sqlite")
	}
}