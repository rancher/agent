package runtime

import (
	"github.com/rancher/agent/utils"
	v3 "github.com/rancher/go-rancher/v3"
	"gopkg.in/check.v1"
)

type ImageInspectTestSuite struct {
}

var _ = check.Suite(&ImageInspectTestSuite{})

func (i *ImageInspectTestSuite) TestImageInspect(c *check.C) {
	testCases := []string{
		"ubuntu:16.04",
		"redis:4.0",
		"rancher/server:v1.6.10",
	}
	expectedResults := []string{
		"sha256:75eae7b7a1ac6cf801996b08abce2f51846a4e0f571d23dc5112f225ae0498fa",
		"sha256:6c021a797fd0d645ab8282c1caaaf7576fa7d91cdce4f9bde2f87839ccff9424",
		"sha256:117c64f21f44ceaac3f4bd1b13c6ce35ed2ec67b7fc692115c9f0ec04819a66e",
	}
	for i, testcase := range testCases {
		credential := v3.Credential{}
		data, err := ImageInspect(testcase, credential)
		if err != nil {
			c.Fatal(err)
		}
		id, ok := utils.GetFieldsIfExist(data, "config", "Image")
		if !ok {
			c.Fail()
		}
		c.Assert(id, check.Equals, expectedResults[i])
	}
}
