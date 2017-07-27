package aws

import (
	"os"
	"path"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/pkg/errors"
	"github.com/rancher/agent/utils"
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

func (f fakeReplyImpl) getInstanceIdentityDocument() (ec2metadata.EC2InstanceIdentityDocument, error) {
	return ec2metadata.EC2InstanceIdentityDocument{Region: "fake", AvailabilityZone: "fake"}, nil
}

type errorReplyImpl struct{}

func (e errorReplyImpl) getInstanceIdentityDocument() (ec2metadata.EC2InstanceIdentityDocument, error) {
	return ec2metadata.EC2InstanceIdentityDocument{}, errors.New("fake error")
}

func (s *ComputeTestSuite) TestGetCloudProviderInfo(c *check.C) {
	os.Mkdir(utils.StateDir(), 0755)
	p := Provider{
		interval:    time.Second,
		expireTime:  time.Second * 5,
		initialized: true,
	}
	p.client = fakeReplyImpl{}
	success := p.GetCloudProviderInfo()
	c.Assert(success, check.Equals, true)
	infoPath := path.Join(utils.StateDir(), infoFile)
	os.Remove(infoPath)

	p.client = errorReplyImpl{}
	success = p.GetCloudProviderInfo()
	c.Assert(success, check.Equals, false)
}
