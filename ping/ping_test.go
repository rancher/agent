package ping

import (
	"gopkg.in/check.v1"
	"github.com/rancher/agent/utils"
	"github.com/docker/docker/api/types/container"
	"testing"
	"golang.org/x/net/context"
	"github.com/docker/docker/api/types"
	v2 "github.com/rancher/go-rancher/v2"
	"github.com/docker/docker/api/types/filters"
	"time"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) {
	check.TestingT(t)
}

type ComputeTestSuite struct {
}

var _ = check.Suite(&ComputeTestSuite{})

func (s *ComputeTestSuite) SetUpSuite(c *check.C) {
}

func (s *ComputeTestSuite) TestNoLabelField(c *check.C) {
	deleteContainer("/no-label-test")
	client := utils.GetRuntimeClient("docker", utils.DefaultVersion)
	config := container.Config{Image: "ibuildthecloud/helloworld:latest"}
	resp, err := client.ContainerCreate(context.Background(), &config, nil, nil, "no-label-test")
	if err != nil {
		c.Fatal(err)
	}
	err = client.ContainerStart(context.Background(), resp.ID, types.ContainerStartOptions{})
	if err != nil {
		c.Fatal(err)
	}
	containerSpec := v2.Container{
		Uuid:       "irrelevant",
		ExternalId: resp.ID,
	}
	containerId, err := utils.FindContainer(client, containerSpec, false)
	if err != nil {
		c.Fatal(err)
	}
	containers := []Resource{}
	filter := filters.NewArgs()
	filter.Add("id", containerId)
	cont, err := client.ContainerList(context.Background(), types.ContainerListOptions{
		Filters: filter,
	})
	if err != nil || len(cont) == 0{
		c.Fatal(err)
	}

	cont[0].Labels = map[string]string{}
	containers = addContainer("running", cont[0], containers)
	c.Assert(containers[0].UUID, check.Equals, "no-label-test")
	c.Assert(containers[0].SystemContainer, check.Equals, "")
}

func deleteContainer(name string) {
	client := utils.GetRuntimeClient("docker", utils.DefaultVersion)
	containerList, _ := client.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	for _, c := range containerList {
		found := false
		labels := c.Labels
		if labels["io.rancher.container.uuid"] == name[1:] {
			found = true
		}

		for _, cname := range c.Names {
			if name == cname {
				found = true
				break
			}
		}
		if found {
			client.ContainerKill(context.Background(), c.ID, "KILL")
			for i := 0; i < 10; i++ {
				if inspect, err := client.ContainerInspect(context.Background(), c.ID); err == nil && inspect.State.Pid == 0 {
					break
				}
				time.Sleep(time.Duration(500) * time.Millisecond)
			}
			client.ContainerRemove(context.Background(), c.ID, types.ContainerRemoveOptions{})
		}
	}
}
