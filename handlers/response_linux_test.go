package handlers

import (
	"gopkg.in/check.v1"
	"github.com/rancher/agent/utils"
	"github.com/docker/docker/api/types/container"
	"golang.org/x/net/context"
	"github.com/docker/docker/api/types"
)

type ResponseTestSuite struct {
}

var _ = check.Suite(&ResponseTestSuite{})

func (s *ResponseTestSuite) SetUpSuite(c *check.C) {
}

func (s *ResponseTestSuite) TestGetIP(c *check.C) {
	client := utils.GetRuntimeClient("docker", utils.DefaultVersion)
	config := container.Config{
		Image: "ibuildthecloud/helloworld:latest",
		Labels: map[string]string{
			UUIDLabel: "c861f990-4472-4fa1-960f-65171b544c29",
		},
	}

	resp, err := client.ContainerCreate(context.Background(), &config, nil, nil, "iptest")
	if err != nil {
		c.Fatal(err)
	}

	err = client.ContainerStart(context.Background(), resp.ID, types.ContainerStartOptions{})
	if err != nil {
		c.Fatal(err)
	}

	inspect, err := client.ContainerInspect(context.Background(), resp.ID)
	if err != nil {
		c.Fatal(err)
	}
	ip, err := getIP(inspect, "bridge", nil)
	if err != nil {
		c.Fatal(err)
	}
	c.Assert(ip, check.Equals, inspect.NetworkSettings.IPAddress)
}
