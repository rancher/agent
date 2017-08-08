//+build !windows

package runtime

import (
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/rancher/agent/utils"
	v3 "github.com/rancher/go-rancher/v3"
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

func (s *ContainerStartTestSuite) TestSetupProxy(c *check.C) {
	instance := v3.Container{System: true}

	// test case 1: test override by host_entries
	config1 := &container.Config{Env: []string{"no_proxy=bar"}}
	setupProxy(instance, config1, map[string]string{"no_proxy": "bar1"})
	c.Assert(config1.Env, check.DeepEquals, []string{"no_proxy=bar1"})

	// test case 2: ignore empty no_proxy value
	config2 := &container.Config{Env: []string{}}
	setupProxy(instance, config2, map[string]string{"no_proxy": ""})
	c.Assert(config2.Env, check.DeepEquals, []string{})

	// test case 3: no-equal case
	config3 := &container.Config{Env: []string{"no_proxy=foo"}}
	setupProxy(instance, config3, map[string]string{})
	c.Assert(config3.Env, check.DeepEquals, []string{"no_proxy=foo"})

	// test case 4: normal case
	config4 := &container.Config{Env: []string{}}
	setupProxy(instance, config4, map[string]string{"no_proxy": "foo"})
	c.Assert(config4.Env, check.DeepEquals, []string{"no_proxy=foo"})

	// test case 5: override no-equal case
	config5 := &container.Config{Env: []string{"no_proxy"}}
	setupProxy(instance, config5, map[string]string{"no_proxy": "foo"})
	c.Assert(config5.Env, check.DeepEquals, []string{"no_proxy=foo"})

	// test case 6: override non-setting
	config6 := &container.Config{Env: []string{"no_proxy", "http_proxy=", "https_proxy=foo"}}
	setupProxy(instance, config6, map[string]string{})
	c.Assert(utils.SearchInList(config6.Env, "no_proxy"), check.Equals, true)
	c.Assert(utils.SearchInList(config6.Env, "http_proxy="), check.Equals, true)
	c.Assert(utils.SearchInList(config6.Env, "https_proxy=foo"), check.Equals, true)
}
