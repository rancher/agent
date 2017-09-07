package aliyun

import (
	"os"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/rancher/agent/cloudprovider"
	"github.com/rancher/agent/utilities/config"

	"gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) {
	check.TestingT(t)
}

type ComputeTestSuite struct {
}

var _ = check.Suite(&ComputeTestSuite{})

func (s *ComputeTestSuite) SetUpSuite(c *check.C) {
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

func (s *ComputeTestSuite) TestGetCloudProviderInfo(c *check.C) {
	os.Mkdir(config.StateDir(), 0755)
	p := Provider{
		interval:    time.Second,
		expireTime:  time.Second * 5,
		initialized: true,
	}
	p.client = fakeReplyImpl{}
	success := p.GetCloudProviderInfo()
	c.Assert(success, check.Equals, true)
	os.Remove(cloudprovider.InfoPath)

	p.client = errorReplyImpl{}
	success = p.GetCloudProviderInfo()
	c.Assert(success, check.Equals, false)
}
