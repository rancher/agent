package ping

import (
	"testing"

	"github.com/cnf/structhash"
	"gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) {
	check.TestingT(t)
}

type PingTestSuite struct {
}

var _ = check.Suite(&PingTestSuite{})

func (s *PingTestSuite) SetUpSuite(c *check.C) {
}

func (s *PingTestSuite) TestHashKey(c *check.C) {
	// put in the same struct, hash should be the same
	compute1 := Resource{
		Type:     "host",
		Kind:     "docker",
		HostName: "rancher.com",
		CreateLabels: map[string]string{
			"foo1": "bar1",
			"foo2": "bar2",
		},
		Labels: map[string]string{
			"foo1": "bar1",
			"foo2": "bar2",
		},
		MachineServiceRegistrationUUID: "uuid",
	}
	compute2 := Resource{
		Type:     "host",
		Kind:     "docker",
		HostName: "rancher.com",
		CreateLabels: map[string]string{
			"foo1": "bar1",
			"foo2": "bar2",
		},
		Labels: map[string]string{
			"foo1": "bar1",
			"foo2": "bar2",
		},
		MachineServiceRegistrationUUID: "uuid",
	}
	hash1 := structhash.Sha1(compute1, 1)
	hash2 := structhash.Sha1(compute2, 1)
	c.Assert(hash1, check.DeepEquals, hash2)

	// change some fields, hash should be different
	compute1.HostName = "rancher2.com"
	hash1 = structhash.Sha1(compute1, 1)
	hash2 = structhash.Sha1(compute2, 1)
	c.Assert(hash1, check.Not(check.DeepEquals), hash2)

	compute1.HostName = "rancher.com"
	compute1.Labels = map[string]string{
		"foo1": "bar1",
		"foo2": "bar3",
	}
	hash1 = structhash.Sha1(compute1, 1)
	hash2 = structhash.Sha1(compute2, 1)
	c.Assert(hash1, check.Not(check.DeepEquals), hash2)

	// add localStorage, hash should still be the same
	compute1.HostName = "rancher.com"
	compute1.Labels = map[string]string{
		"foo1": "bar1",
		"foo2": "bar2",
	}
	compute1.LocalStorageMb = 10000
	compute2.LocalStorageMb = 10001
	hash1 = structhash.Sha1(compute1, 1)
	hash2 = structhash.Sha1(compute2, 1)
	c.Assert(hash1, check.DeepEquals, hash2)
}
