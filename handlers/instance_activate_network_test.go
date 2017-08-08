package handlers

import (
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/rancher/agent/utils"
	v3 "github.com/rancher/go-rancher/v3"
	"golang.org/x/net/context"
	"gopkg.in/check.v1"
)

func (s *EventTestSuite) TestNetworkModeNone(c *check.C) {
	deleteContainer("85db87bf-cb14-4643-9e7d-a13e3e77a991")

	var request v3.DeploymentSyncRequest
	event := getDeploymentSyncRequest("./test_events/deployment_sync_request", &request, c)
	c.Assert(request.Containers, check.HasLen, 1)

	request.Networks = []v3.Network{{Kind: "dockerNone", Resource: v3.Resource{Id: "1n5"}}}
	request.Containers[0].Hostname = "nameisset"
	request.Containers[0].PrimaryNetworkId = "1n5"

	event.Data["deploymentSyncRequest"] = request
	rawEvent := marshalEvent(event, c)
	reply := testEvent(rawEvent, c)

	c.Assert(reply.Transitioning != "error", check.Equals, true)

	inspect := getDockerInspect(reply, c)
	c.Assert(inspect.HostConfig.NetworkMode.IsNone(), check.Equals, true)
	c.Assert(inspect.Config.NetworkDisabled, check.Equals, true)
	c.Assert(inspect.Config.Hostname, check.Equals, "nameisset")
}

func (s *EventTestSuite) TestNetworkModeHost(c *check.C) {
	deleteContainer("85db87bf-cb14-4643-9e7d-a13e3e77a991")

	var request v3.DeploymentSyncRequest
	event := getDeploymentSyncRequest("./test_events/deployment_sync_request", &request, c)
	c.Assert(request.Containers, check.HasLen, 1)

	request.Networks = []v3.Network{{Kind: "dockerHost", Resource: v3.Resource{Id: "1n5"}}}
	request.Containers[0].Hostname = "nameisset"
	request.Containers[0].PrimaryNetworkId = "1n5"

	event.Data["deploymentSyncRequest"] = request
	rawEvent := marshalEvent(event, c)
	reply := testEvent(rawEvent, c)

	c.Assert(reply.Transitioning != "error", check.Equals, true)

	inspect := getDockerInspect(reply, c)
	c.Assert(inspect.HostConfig.NetworkMode.IsHost(), check.Equals, true)
	c.Assert(inspect.Config.NetworkDisabled, check.Equals, false)
	c.Assert(inspect.Config.Hostname != "nameisset", check.Equals, true)
}

func (s *EventTestSuite) TestNetworkModeContainer(c *check.C) {
	deleteContainer("85db87bf-cb14-4643-9e7d-a13e3e77a991")
	deleteContainer("/network_container")

	cli := utils.GetRuntimeClient("docker", utils.DefaultVersion)
	config := &container.Config{
		Image:     "ibuildthecloud/helloworld:latest",
		OpenStdin: true,
	}
	hostConfig := &container.HostConfig{}
	resp, err := cli.ContainerCreate(context.Background(), config, hostConfig, nil, "network_container")
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

	request.Networks = []v3.Network{{Kind: "dockerContainer", Resource: v3.Resource{Id: "1n5"}}}
	request.Containers[0].Hostname = "notset"
	request.Containers[0].PrimaryNetworkId = "1n5"
	request.Containers[0].NetworkContainerId = "1c1"
	request.Containers = append(request.Containers, v3.Container{ExternalId: resp.ID, Resource: v3.Resource{Id: "1c1"}})

	event.Data["deploymentSyncRequest"] = request
	rawEvent := marshalEvent(event, c)
	reply := testEvent(rawEvent, c)

	c.Assert(reply.Transitioning != "error", check.Equals, true)

	inspect := getDockerInspect(reply, c)
	c.Assert(inspect.HostConfig.NetworkMode.IsContainer(), check.Equals, true)
	c.Assert(inspect.HostConfig.NetworkMode, check.Equals, container.NetworkMode(fmt.Sprintf("container:%v", resp.ID)))
	c.Assert(inspect.Config.NetworkDisabled, check.Equals, false)
	c.Assert(inspect.Config.Hostname != "notset", check.Equals, true)
}

func (s *EventTestSuite) TestNetworkModeBridge(c *check.C) {
	deleteContainer("85db87bf-cb14-4643-9e7d-a13e3e77a991")

	var request v3.DeploymentSyncRequest
	event := getDeploymentSyncRequest("./test_events/deployment_sync_request", &request, c)
	c.Assert(request.Containers, check.HasLen, 1)

	request.Networks = []v3.Network{{Kind: "dockerBridge", Resource: v3.Resource{Id: "1n5"}}}
	request.Containers[0].Hostname = "nameisset"
	request.Containers[0].PrimaryNetworkId = "1n5"
	request.Containers[0].PublicEndpoints = []v3.PublicEndpoint{
		{
			PublicPort:  10003,
			PrivatePort: 10000,
			Protocol:    "tcp",
		},
	}

	event.Data["deploymentSyncRequest"] = request
	rawEvent := marshalEvent(event, c)
	reply := testEvent(rawEvent, c)

	c.Assert(reply.Transitioning != "error", check.Equals, true)

	inspect := getDockerInspect(reply, c)
	c.Assert(inspect.HostConfig.NetworkMode.IsDefault(), check.Equals, true)
	c.Assert(inspect.Config.NetworkDisabled, check.Equals, false)
	c.Assert(inspect.Config.Hostname, check.Equals, "nameisset")
	c.Assert(inspect.HostConfig.PortBindings["10000/tcp"][0], check.Equals, nat.PortBinding{HostPort: "10003"})
}
