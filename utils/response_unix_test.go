//+build !windows

package utils

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/patrickmn/go-cache"
	"github.com/rancher/agent/utils"
	"gopkg.in/check.v1"
	"testing"
	"time"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) {
	check.TestingT(t)
}

type UtilTestSuite struct {
}

var _ = check.Suite(&UtilTestSuite{})

func (s *UtilTestSuite) SetUpSuite(c *check.C) {
}


