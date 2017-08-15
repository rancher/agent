//+build !windows

package utils

import (
	"fmt"
	"os"

	"github.com/nu7hatch/gouuid"
	"gopkg.in/check.v1"
	"testing"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

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
