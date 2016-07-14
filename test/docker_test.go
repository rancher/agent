package test

import (
	"github.com/Sirupsen/logrus"
	"github.com/mitchellh/mapstructure"
	"github.com/rancher/agent/handlers/dockerClient"
	"github.com/rancher/agent/handlers/progress"
	"github.com/rancher/agent/handlers/utils"
	"github.com/rancher/agent/model"
	"golang.org/x/net/context"
	"gopkg.in/check.v1"
)

func (s *ComputeTestSuite) TestInstanceActivateNoName(c *check.C) {
	logrus.Info("testing InstanceActivateNoName")
	utils.DeleteContainer("/c861f990-4472-4fa1-960f-65171b544c28")
	rawEvent := loadEvent("../test_events/instance_activate", c)
	event := unmarshalEvent(rawEvent, c)
	instance := getInstance(event, c)
	instance["name"] = ""
	rawEvent = marshalEvent(event, c)

	testEvent(rawEvent, c)
	client := dockerClient.GetClient(utils.DefaultVersion)
	var in model.Instance
	mapstructure.Decode(instance, &in)
	container := utils.GetContainer(client, &in, false)
	containerResp, err := client.ContainerInspect(context.Background(), container.ID)
	c.Assert(err, check.IsNil)
	c.Check(containerResp.Name, check.Equals, "/c861f990-4472-4fa1-960f-65171b544c28")
}

func (s *ComputeTestSuite) TestInstanceActivateDuplicateName(c *check.C) {
	logrus.Info("testing InstanceActivateDuplicateName")
	utils.DeleteContainer("/c861f990-4472-4fa1-960f-65171b544c28")
	dupeNameUUID := "dupename-c861f990-4472-4fa1-960f-65171b544c28"
	utils.DeleteContainer("/" + dupeNameUUID)

	rawEvent := loadEvent("../test_events/instance_activate", c)
	event := unmarshalEvent(rawEvent, c)
	instance := getInstance(event, c)
	client := dockerClient.GetClient(utils.DefaultVersion)
	var in model.Instance

	testEvent(rawEvent, c)
	mapstructure.Decode(instance, &in)
	container := utils.GetContainer(client, &in, false)
	stopErr := utils.DoInstanceDeactivate(&in, &progress.Progress{})
	if stopErr != nil {
		logrus.Error(stopErr)
	}

	instance["uuid"] = dupeNameUUID
	rawEvent = marshalEvent(event, c)

	testEvent(rawEvent, c)
	mapstructure.Decode(instance, &in)
	container = utils.GetContainer(client, &in, false)
	containerResp, err := client.ContainerInspect(context.Background(), container.ID)
	c.Assert(err, check.IsNil)
	c.Check(containerResp.Config.Labels["io.rancher.container.uuid"], check.Equals, dupeNameUUID)
	c.Check(containerResp.Name, check.Equals, "/"+dupeNameUUID)
}
