//+build !windows

package utils

import (
	"fmt"
	"os"

	"gopkg.in/check.v1"
	gofqdn "github.com/ShowMax/go-fqdn"
	"github.com/nu7hatch/gouuid"
)

type ConfigTestSuite struct {
}

var _ = check.Suite(&ConfigTestSuite{})

func (s *ConfigTestSuite) SetUpSuite(c *check.C) {
}

func (s *ConfigTestSuite) TestLabels(c *check.C) {
	ConfigOverride["HOST_LABELS"] = "foo=bar&test=1&foo=dontpick&novalue"
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

func (s *ConfigTestSuite) TestDefaultValue(c *check.C) {
	varName, _ := uuid.NewV4()
	cattleVarName := fmt.Sprintf("CATTLE_%v", varName)
	def := "defaulted"
	actual := DefaultValue(varName.String(), def)
	c.Assert(def, check.Equals, actual)

	actual = DefaultValue(varName.String(), "")
	c.Assert(actual, check.Equals, "")

	os.Setenv(cattleVarName, "")
	actual = DefaultValue(varName.String(), def)
	c.Assert(actual, check.Equals, def)

	os.Setenv(cattleVarName, "foobar")
	actual = DefaultValue(varName.String(), def)
	c.Assert(actual, check.Equals, "foobar")

	SetSecretKey("supersecretkey")
	actual = DefaultValue("SECRET_KEY", def)
	c.Assert(actual, check.Equals, "supersecretkey")
}
