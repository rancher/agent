package config

import (
	"testing"

	"gopkg.in/check.v1"

	"github.com/rancher/agent/utilities/constants"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) {
	check.TestingT(t)
}

type ConfigTestSuite struct {
}

var _ = check.Suite(&ConfigTestSuite{})

func (s *ConfigTestSuite) SetUpSuite(c *check.C) {
}

func (s *ConfigTestSuite) TestLabels(c *check.C) {
	constants.ConfigOverride["HOST_LABELS"] = "foo=bar&test=1&foo=dontpick&novalue"
	labels := Labels()
	c.Assert(labels, check.DeepEquals, map[string]string{"foo": "bar", "test": "1", "novalue": ""})

}
