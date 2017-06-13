//+build !windows

package storage

import (
	"github.com/rancher/agent/model"
	"github.com/rancher/agent/utilities/docker"
	"gopkg.in/check.v1"
	"testing"
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

func (s *ComputeTestSuite) TestDoImageActivate(c *check.C) {
	imageUUID := "docker:badimage"
	client := docker.GetClient(docker.DefaultVersion)
	err := PullImage(nil, client, imageUUID, model.BuildOptions{}, model.RegistryCredential{})
	c.Check(err, check.NotNil)
}
