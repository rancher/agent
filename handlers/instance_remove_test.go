package handlers

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/rancher/agent/utils"
	v2 "github.com/rancher/go-rancher/v2"
	"gopkg.in/check.v1"
)

func (s *EventTestSuite) TestInstanceRemove(c *check.C) {
	deleteContainer("85db87bf-cb14-4643-9e7d-a13e3e77a991")

	var request v2.DeploymentSyncRequest
	event := getDeploymentSyncRequest("./test_events/deployment_sync_request", &request, c)
	c.Assert(request.Containers, check.HasLen, 1)

	event.Data["deploymentSyncRequest"] = request
	rawEvent := marshalEvent(event, c)
	reply := testEvent(rawEvent, c)

	c.Assert(reply.Transitioning != "error", check.Equals, true)

	inspect := getDockerInspect(reply, c)
	containerID := inspect.ID

	event.Name = "compute.instance.remove"

	rawEvent = marshalEvent(event, c)
	reply = testEvent(rawEvent, c)

	c.Assert(reply.Transitioning != "error", check.Equals, true)

	client := utils.GetRuntimeClient("docker", utils.DefaultVersion)
	filt := filters.NewArgs()
	filt.Add("id", containerID)
	list, err := client.ContainerList(context.Background(), types.ContainerListOptions{
		All:     true,
		Filters: filt,
	})
	if err != nil {
		c.Fatal(err)
	}
	c.Assert(list, check.HasLen, 0)

	reply = testEvent(rawEvent, c)

	c.Assert(reply.Transitioning != "error", check.Equals, true)
}
