package handlers

import (
	"fmt"
	"github.com/docker/docker/api/types"
	con "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/rancher/agent/utils/docker"
	"github.com/rancher/agent/utils/utils"
	"golang.org/x/net/context"
	"gopkg.in/check.v1"
)

func (s *ComputeTestSuite) TestConflictVolumeRemove(c *check.C) {
	deleteContainer("/c861f990-4472-4fa1-960f-65171b544c28")

	rawEvent := loadEvent("./test_events/instance_activate_volume", c)
	event, _, _ := unmarshalEventAndInstanceFields(rawEvent, c)

	rawEvent = marshalEvent(event, c)
	reply := testEvent(rawEvent, c)

	insp, ok := utils.GetFieldsIfExist(reply.Data, "instance", "+data", "dockerInspect")
	if !ok {
		c.Fatal("No id found")
	}
	inspect := insp.(types.ContainerJSON)
	dockerClient := docker.GetClient(docker.DefaultVersion)
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

	rawEvent2 := loadEvent("./test_events/instance_remove", c)
	event2, _, _ := unmarshalEventAndInstanceFields(rawEvent2, c)

	rawEvent = marshalEvent(event2, c)
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
