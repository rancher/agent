package aliyun

import (
	"os"
	"testing"
	"time"

	"github.com/pkg/errors"
	. "gopkg.in/check.v1"

	"github.com/rancher/agent/cloudprovider"
	"github.com/rancher/agent/core/hostInfo"
	"github.com/rancher/agent/utilities/config"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) {
	TestingT(t)
}

type ComputeTestSuite struct {
}

var _ = Suite(&ComputeTestSuite{})

func (s *ComputeTestSuite) SetUpSuite(c *C) {
}

type fakeReplyImpl struct{}

func (f fakeReplyImpl) Region() (string, error) {
	return "fake", nil
}

func (f fakeReplyImpl) Zone() (string, error) {
	return "fake", nil
}

type errorReplyImpl struct{}

func (e errorReplyImpl) Region() (string, error) {
	return "", errors.New("fake error")
}

func (e errorReplyImpl) Zone() (string, error) {
	return "", errors.New("fake error")
}

func (s *ComputeTestSuite) TestGetHostInfo(c *C) {
	os.Mkdir(config.StateDir(), 0755)
	p := Provider{
		interval:   time.Second,
		expireTime: time.Second * 5,
	}

	i := &hostInfo.Info{}
	i.Labels = map[string]string{
		cloudprovider.RegionLabel:           "fake",
		cloudprovider.AvailabilityZoneLabel: "fake",
		cloudprovider.CloudProviderLabel:    aliyunTag,
	}

	p.client = fakeReplyImpl{}
	hostInfo, err := p.GetHostInfo()
	c.Assert(err, IsNil)
	c.Assert(hostInfo, DeepEquals, i)

	p.client = errorReplyImpl{}
	hostInfo, err = p.GetHostInfo()
	c.Assert(err, ErrorMatches, "fake error")
}
