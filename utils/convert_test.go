package utils

import (
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/blkiodev"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	v3 "github.com/rancher/go-rancher/v3"
	"gopkg.in/check.v1"
)

type ConvertTestSuite struct {
}

var _ = check.Suite(&ConvertTestSuite{})

func (s *ConvertTestSuite) SetUpSuite(c *check.C) {
}

func (s *ConvertTestSuite) TestConvertBlkioOptions(c *check.C) {
	inspect := types.ContainerJSON{ContainerJSONBase: &types.ContainerJSONBase{}}
	inspect.HostConfig = &container.HostConfig{}
	inspect.HostConfig.BlkioWeightDevice = []*blkiodev.WeightDevice{
		{
			Path:   "/dev/null",
			Weight: 1000,
		},
		{
			Path:   "/dev/sda",
			Weight: 2000,
		},
	}
	inspect.HostConfig.BlkioDeviceWriteIOps = []*blkiodev.ThrottleDevice{
		{
			Path: "/dev/sda1",
			Rate: 1000,
		},
	}
	inspect.HostConfig.BlkioDeviceReadBps = []*blkiodev.ThrottleDevice{
		{
			Path: "/dev/sda2",
			Rate: 1000,
		},
	}
	inspect.HostConfig.BlkioDeviceReadIOps = []*blkiodev.ThrottleDevice{
		{
			Path: "/dev/sda3",
			Rate: 1000,
		},
	}
	inspect.HostConfig.BlkioDeviceWriteBps = []*blkiodev.ThrottleDevice{
		{
			Path: "/dev/sda4",
			Rate: 1000,
		},
	}
	result := convertBlkioOptions(inspect)
	expected := map[string]interface{}{
		"/dev/null": map[string]interface{}{
			"weight": int64(1000),
		},
		"/dev/sda": map[string]interface{}{
			"weight": int64(2000),
		},
		"/dev/sda1": map[string]interface{}{
			"writeIops": int64(1000),
		},
		"/dev/sda2": map[string]interface{}{
			"readBps": int64(1000),
		},
		"/dev/sda3": map[string]interface{}{
			"readIops": int64(1000),
		},
		"/dev/sda4": map[string]interface{}{
			"writeBps": int64(1000),
		},
	}
	c.Assert(result, check.DeepEquals, expected)
}

func (s *ConvertTestSuite) TestConvertDevice(c *check.C) {
	inspect := types.ContainerJSON{ContainerJSONBase: &types.ContainerJSONBase{}}
	inspect.HostConfig = &container.HostConfig{}
	inspect.HostConfig.Devices = []container.DeviceMapping{
		{
			PathInContainer:   "/dev/null",
			PathOnHost:        "/dev/null",
			CgroupPermissions: "rw",
		},
		{
			PathInContainer: "/dev/sda",
			PathOnHost:      "/dev/sda",
		},
	}
	result := convertDevice(inspect)
	c.Assert(result, check.DeepEquals, []string{"/dev/null:/dev/null:rw", "/dev/sda:/dev/sda:rwm"})
}

func (s *ConvertTestSuite) TestConvertPublicEndpoint(c *check.C) {
	inspect := types.ContainerJSON{ContainerJSONBase: &types.ContainerJSONBase{}}
	inspect.HostConfig = &container.HostConfig{}
	_, bindings, err := nat.ParsePortSpecs([]string{
		"127.0.0.1:8080:8080/tcp",
		"8081:8081/udp",
	})
	if err != nil {
		c.Fatal(err)
	}
	inspect.HostConfig.PortBindings = bindings
	result := convertPublicEndpoint(inspect)
	for _, r := range result {
		if r.Protocol == "tcp" {
			c.Assert(r, check.DeepEquals, v3.PublicEndpoint{
				IpAddress:   "127.0.0.1",
				PrivatePort: int64(8080),
				PublicPort:  int64(8080),
				Protocol:    "tcp",
			})
		} else if r.Protocol == "udp" {
			c.Assert(r, check.DeepEquals, v3.PublicEndpoint{
				IpAddress:   "",
				PrivatePort: int64(8081),
				PublicPort:  int64(8081),
				Protocol:    "udp",
			})
		} else {
			c.Fail()
		}
	}
}
