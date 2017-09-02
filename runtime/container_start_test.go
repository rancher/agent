//+build !windows

package runtime

import (
	"testing"

	"gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) {
	check.TestingT(t)
}

type ContainerStartTestSuite struct {
}

var _ = check.Suite(&ContainerStartTestSuite{})

func (s *ContainerStartTestSuite) SetUpSuite(c *check.C) {
}
