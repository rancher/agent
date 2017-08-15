package handlers

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/rancher/agent/utils"
	v3 "github.com/rancher/go-rancher/v3"
	"gopkg.in/check.v1"
)

// Recieving an activate event for a running, pre-existing container should
// result in the container continuing to run and the appropriate data sent
// back in the response (like, ports, ip, inspect, etc)
func (s *EventTestSuite) TestNativeInstanceActivateOnly(c *check.C) {
	deleteContainer("/native_container")

	cli := utils.GetRuntimeClient("docker", utils.DefaultVersion)
	config := &container.Config{
		Image:     "ibuildthecloud/helloworld:latest",
		OpenStdin: true,
	}
	hostConfig := &container.HostConfig{}
	resp, err := cli.ContainerCreate(context.Background(), config, hostConfig, nil, "native_container")
	if err != nil {
		c.Fatal(err)
	}
	err = cli.ContainerStart(context.Background(), resp.ID, types.ContainerStartOptions{})
	if err != nil {
		c.Fatal(err)
	}

	var request v3.DeploymentSyncRequest
	event := getDeploymentSyncRequest("./test_events/deployment_sync_request", &request, c)
	c.Assert(request.Containers, check.HasLen, 1)

	request.Containers[0].ExternalId = resp.ID
	event.Data["deploymentSyncRequest"] = request

	rawEvent := marshalEvent(event, c)
	reply := testEvent(rawEvent, c)

	c.Assert(reply.Transitioning != "error", check.Equals, true)

	inspect := getContainerSpec(reply, c)
	c.Assert(inspect.ExternalId, check.Equals, resp.ID)
	insp := inspectContainer(resp.ID, c)
	c.Assert(insp.State.Running, check.Equals, true)
}

// Receiving an activate event for a pre-existing stopped container
// that Rancher never recorded as having started should result in the
// container staying stopped.
func (s *EventTestSuite) TestNativeInstanceNotRunning(c *check.C) {
	deleteContainer("/native_container")

	cli := utils.GetRuntimeClient("docker", utils.DefaultVersion)
	config := &container.Config{
		Image:     "ibuildthecloud/helloworld:latest",
		OpenStdin: true,
	}
	hostConfig := &container.HostConfig{}
	resp, err := cli.ContainerCreate(context.Background(), config, hostConfig, nil, "native_container")
	if err != nil {
		c.Fatal(err)
	}

	var request v3.DeploymentSyncRequest
	event := getDeploymentSyncRequest("./test_events/deployment_sync_request", &request, c)
	c.Assert(request.Containers, check.HasLen, 1)

	request.Containers[0].ExternalId = resp.ID
	event.Data["deploymentSyncRequest"] = request
	event.Data["processData"] = map[string]interface{}{
		"containerNoOpEvent": true,
	}

	rawEvent := marshalEvent(event, c)
	reply := testEvent(rawEvent, c)

	c.Assert(reply.Transitioning != "error", check.Equals, true)

	inspect := getContainerSpec(reply, c)
	c.Assert(inspect.ExternalId, check.Equals, resp.ID)
	insp := inspectContainer(resp.ID, c)
	c.Assert(insp.State.Running, check.Equals, false)
}

// Receiving an activate event for a pre-existing, but removed container
// should result in the container continuing to not exist and a valid but
// minimally populated response.
func (s *EventTestSuite) TestNativeInstanceRemoved(c *check.C) {
	deleteContainer("/native_container")

	cli := utils.GetRuntimeClient("docker", utils.DefaultVersion)
	config := &container.Config{
		Image:     "ibuildthecloud/helloworld:latest",
		OpenStdin: true,
	}
	hostConfig := &container.HostConfig{}
	resp, err := cli.ContainerCreate(context.Background(), config, hostConfig, nil, "native_container")
	if err != nil {
		c.Fatal(err)
	}
	deleteContainer("/native_container")

	var request v3.DeploymentSyncRequest
	event := getDeploymentSyncRequest("./test_events/deployment_sync_request", &request, c)
	c.Assert(request.Containers, check.HasLen, 1)

	request.Containers[0].ExternalId = resp.ID
	event.Data["deploymentSyncRequest"] = request
	event.Data["processData"] = map[string]interface{}{
		"containerNoOpEvent": true,
	}

	rawEvent := marshalEvent(event, c)
	reply := testEvent(rawEvent, c)

	c.Assert(reply.Transitioning != "error", check.Equals, true)
}

func (s *EventTestSuite) TestNativeInstanceDeactivateOnly(c *check.C) {
	s.TestNativeInstanceActivateOnly(c)

	cont := findContainer("/native_container")
	var request v3.DeploymentSyncRequest
	event := getDeploymentSyncRequest("./test_events/deployment_sync_request", &request, c)
	c.Assert(request.Containers, check.HasLen, 1)

	event.Name = "compute.instance.deactivate"
	request.Containers[0].ExternalId = cont.ID
	event.Data["deploymentSyncRequest"] = request

	rawEvent := marshalEvent(event, c)
	reply := testEvent(rawEvent, c)

	c.Assert(reply.Transitioning != "error", check.Equals, true)
	insp := inspectContainer(cont.ID, c)
	c.Assert(insp.State.Running, check.Equals, false)

	event = getDeploymentSyncRequest("./test_events/deployment_sync_request", &request, c)
	c.Assert(request.Containers, check.HasLen, 1)
	request.Containers[0].ExternalId = cont.ID

	event.Data["deploymentSyncRequest"] = request
	rawEvent = marshalEvent(event, c)
	reply = testEvent(rawEvent, c)

	c.Assert(reply.Transitioning != "error", check.Equals, true)
	insp = inspectContainer(cont.ID, c)
	c.Assert(insp.State.Running, check.Equals, true)
}

// If a container receives a no-op deactivate event, it should not
// be deactivated.
func (s *EventTestSuite) TestNativeInstanceDeactivateNoOp(c *check.C) {
	s.TestNativeInstanceActivateOnly(c)

	cont := findContainer("/native_container")

	var request v3.DeploymentSyncRequest
	event := getDeploymentSyncRequest("./test_events/deployment_sync_request", &request, c)
	c.Assert(request.Containers, check.HasLen, 1)

	request.Containers[0].ExternalId = cont.ID
	event.Data["deploymentSyncRequest"] = request
	event.Name = "compute.instance.deactivate"
	event.Data["processData"] = map[string]interface{}{
		"containerNoOpEvent": true,
	}

	rawEvent := marshalEvent(event, c)
	reply := testEvent(rawEvent, c)

	inspect := getContainerSpec(reply, c)
	c.Assert(reply.Transitioning != "error", check.Equals, true)
	c.Assert(inspect.ExternalId, check.Equals, cont.ID)
	insp := inspectContainer(cont.ID, c)
	c.Assert(insp.State.Running, check.Equals, true)
}
