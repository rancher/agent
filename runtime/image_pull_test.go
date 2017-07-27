//+build !windows

package runtime

import (
	v2 "github.com/rancher/go-rancher/v2"
	"github.com/rancher/agent/utils"
	"gopkg.in/check.v1"
)

var _ = check.Suite(&ComputeTestSuite{})

func (s *ComputeTestSuite) TestDoImageActivate(c *check.C) {
	imageUUID := "docker:badimage"
	client := utils.GetRuntimeClient("docker", utils.DefaultVersion)
	err := ImagePull(nil, client, imageUUID, &v2.DockerBuild{}, RegistryCredential{})
	c.Check(err, check.NotNil)
}
