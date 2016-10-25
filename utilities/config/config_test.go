//+build !windows

package config

import (
	"testing"

	"gopkg.in/check.v1"

	gofqdn "github.com/ShowMax/go-fqdn"
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

func (s *ConfigTestSuite) unTestHostName(c *check.C) {
	// by default getFQDNLinux should just have the same with getFQDNByIP
	fqdn1, err := getFQDNLinux()
	if err != nil {
		c.Fatal(err)
	}
	fqdn2 := gofqdn.Get()
	c.Assert(fqdn1, check.Equals, fqdn2)
}
