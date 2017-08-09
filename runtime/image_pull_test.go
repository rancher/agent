//+build !windows

package runtime

import (
	"github.com/rancher/agent/utils"
	v3 "github.com/rancher/go-rancher/v3"
	"gopkg.in/check.v1"
)

type ImagePullTestSuite struct {
}

var _ = check.Suite(&ContainerStartTestSuite{})

func (s *ImagePullTestSuite) TestDoImageActivate(c *check.C) {
	imageUUID := "docker:badimage"
	client := utils.GetRuntimeClient("docker", utils.DefaultVersion)
	err := ImagePull(nil, client, imageUUID, v3.Credential{})
	c.Check(err, check.NotNil)
}
