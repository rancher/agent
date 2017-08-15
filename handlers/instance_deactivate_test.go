package handlers

import (
	v3 "github.com/rancher/go-rancher/v3"
	"gopkg.in/check.v1"
)

func (s *EventTestSuite) TestInstanceDeactivate(c *check.C) {
	deleteContainer("85db87bf-cb14-4643-9e7d-a13e3e77a991")

	var request v3.DeploymentSyncRequest
	event := getDeploymentSyncRequest("./test_events/deployment_sync_request", &request, c)
	c.Assert(request.Containers, check.HasLen, 1)

	event.Data["deploymentSyncRequest"] = request
	rawEvent := marshalEvent(event, c)
	reply := testEvent(rawEvent, c)

	c.Assert(reply.Transitioning != "error", check.Equals, true)

	event.Name = "compute.instance.deactivate"

	rawEvent = marshalEvent(event, c)
	reply = testEvent(rawEvent, c)

	c.Assert(reply.Transitioning != "error", check.Equals, true)

	inspect := getContainerSpec(reply, c)

	insp := inspectContainer(inspect.ExternalId, c)
	c.Assert(insp.State.Pid, check.Equals, 0)
}
