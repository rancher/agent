//+build !windows

package storage

import (
	"github.com/rancher/agent/model"
	"github.com/rancher/agent/utilities/constants"
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
	image := model.Image{}
	storagePool := model.StoragePool{}
	imageUUID := "docker:badimage"
	client := docker.GetClient(constants.DefaultVersion)
	err := DoImageActivate(image, storagePool, nil, client, imageUUID)
	c.Check(err, check.NotNil)
}

func (s *ComputeTestSuite) TestPullingPrivateImage(c *check.C) {
	image := model.Image{
		RegistryCredential: model.RegistryCredential{
			PublicValue: "strongmonkey1992",
			SecretValue: "pds123456",
			Data: model.CredentialData{
				Fields: model.CredentialFields{
					ServerAddress: "https://index.docker.io/v1",
					Email:         "daishan1992@gmail.com",
				},
			},
		},
	}
	dclient := docker.GetClient(constants.DefaultVersion)
	imageUUID := "strongmonkey1992/docker-whale"
	storagePool := model.StoragePool{}
	err := DoImageActivate(image, storagePool, nil, dclient, imageUUID)
	if err != nil {
		c.Fatal(err)
	}
}
