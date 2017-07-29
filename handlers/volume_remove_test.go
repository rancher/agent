package handlers

import (
	"fmt"

	con "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/rancher/agent/utils"
	v2 "github.com/rancher/go-rancher/v2"
	"golang.org/x/net/context"
	"gopkg.in/check.v1"
)

func (s *EventTestSuite) unTestConflictVolumeRemove(c *check.C) {
	deleteContainer("85db87bf-cb14-4643-9e7d-a13e3e77a991")

	var request v2.DeploymentSyncRequest
	event := getDeploymentSyncRequest("./test_events/deployment_sync_request", &request, c)
	c.Assert(request.Containers, check.HasLen, 1)

	request.Containers[0].DataVolumes = []string{"/foo"}

	event.Data["deploymentSyncRequest"] = request
	rawEvent := marshalEvent(event, c)
	reply := testEvent(rawEvent, c)

	c.Assert(reply.Transitioning != "error", check.Equals, true)

	inspect := getDockerInspect(reply, c)

	dockerClient := utils.GetRuntimeClient("docker", utils.DefaultVersion)
	volumeName := ""
	volume := ""
	for _, mounts := range inspect.Mounts {
		if mounts.Destination == "/foo" {
			volumeName = string(mounts.Name)
			volume = fmt.Sprintf("%v:%v", string(mounts.Name), "/bar")
		}
	}
	config := &con.Config{
		Image:     "ibuildthecloud/helloworld:latest",
		OpenStdin: true,
	}
	hostConfig := &con.HostConfig{}
	config.Volumes = map[string]struct{}{
		volume: {},
	}
	_, err := dockerClient.ContainerCreate(context.Background(), config, hostConfig, nil, "test")
	if err != nil {
		c.Fatal(err)
	}

	event.Name = "compute.instance.remove"

	rawEvent = marshalEvent(event, c)
	reply = testEvent(rawEvent, c)
	c.Assert(reply.Transitioning, check.Not(check.Equals), "error")

	rawEvent3 := loadEvent("./test_events/volume_remove", c)
	event3 := unmarshalEvent(rawEvent3, c)
	event3["data"].(map[string]interface{})["volume"].(map[string]interface{})["name"] = volume
	rawEvent3 = marshalEvent(event3, c)
	reply = testEvent(rawEvent3, c)
	c.Assert(reply.Transitioning, check.Not(check.Equals), "error")
	// check if the volume still exists because of our flaky logic
	filter := filters.NewArgs()
	list, err := dockerClient.VolumeList(context.Background(), filter)
	if err != nil {
		c.Fatal(err)
	}
	found := false
	for _, vo := range list.Volumes {
		if vo.Name == volumeName {
			found = true
		}
	}
	c.Assert(found, check.Equals, true)
}
