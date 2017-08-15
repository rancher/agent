package handlers

import (
	"github.com/rancher/agent/ping"
	v3 "github.com/rancher/go-rancher/v3"
	"gopkg.in/check.v1"
)

func (s *EventTestSuite) TestPing(c *check.C) {
	deleteContainer("85db87bf-cb14-4643-9e7d-a13e3e77a991")

	var request v3.DeploymentSyncRequest
	event := getDeploymentSyncRequest("./test_events/deployment_sync_request", &request, c)
	c.Assert(request.Containers, check.HasLen, 1)

	event.Data["deploymentSyncRequest"] = request
	rawEvent := marshalEvent(event, c)
	reply := testEvent(rawEvent, c)

	c.Assert(reply.Transitioning != "error", check.Equals, true)

	inspect := getDockerInspect(reply, c)
	containerID := inspect.ID

	pingEvent := loadEvent("./test_events/ping", c)
	reply = testEvent(pingEvent, c)

	c.Assert(reply.Transitioning != "error", check.Equals, true)

	resources := reply.Data["resources"].([]ping.Resource)

	// first one is compute
	c.Assert(resources[0].Type, check.Equals, "host")
	c.Assert(resources[0].Kind, check.Equals, "docker")
	c.Assert(resources[0].HostName != "", check.Equals, true)
	//c.Assert(resources[1].Labels["io.rancher.host.docker_version"] != "", check.Equals, true)
	//c.Assert(resources[1].Labels["io.rancher.host.linux_kernel_version"] != "", check.Equals, true)
	c.Assert(len(resources[0].Info), check.Equals, 7)

	// second one is storage
	c.Assert(resources[1].Type, check.Equals, "storagePool")

	//third one is ip
	c.Assert(resources[2].Type, check.Equals, "ipAddress")

	//the rest are containers
	exist1 := false
	for i := 3; i < len(resources); i++ {
		c.Assert(resources[i].Type, check.Equals, "instance")
		if resources[i].ExternalID == containerID && resources[i].State == "running" {
			exist1 = true
		}
	}
	c.Assert(exist1, check.Equals, true)

	c.Assert(reply.Data["hashKey"].(string) != "", check.Equals, true)
}
