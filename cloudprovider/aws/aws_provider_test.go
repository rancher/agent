package aws

import (
	"os"
	"testing"
	"time"
	. "gopkg.in/check.v1"

	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/pkg/errors"

	"github.com/rancher/agent/cloudprovider"
	"github.com/rancher/agent/host_info"
	"github.com/rancher/agent/utils"
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

func (f fakeReplyImpl) getInstanceIdentityDocument() (ec2metadata.EC2InstanceIdentityDocument, error) {
	return ec2metadata.EC2InstanceIdentityDocument{Region: "fake", AvailabilityZone: "fake"}, nil
}

type errorReplyImpl struct{}

func (e errorReplyImpl) getInstanceIdentityDocument() (ec2metadata.EC2InstanceIdentityDocument, error) {
	return ec2metadata.EC2InstanceIdentityDocument{}, errors.New("fake error")
}

func (s *ComputeTestSuite) TestGetHostInfo(c *C) {
	os.Mkdir(utils.StateDir(), 0755)
	p := Provider{
		interval:   time.Second,
		retryCount: 2,
	}
	i := &hostInfo.Info{}
	i.Labels = map[string]string{
		cloudprovider.RegionLabel:           "fake",
		cloudprovider.AvailabilityZoneLabel: "fake",
		cloudprovider.CloudProviderLabel:    awsTag,
	}

	p.client = fakeReplyImpl{}
	hostInfo, err := p.GetHostInfo()
	c.Assert(err, IsNil)
	c.Assert(hostInfo, DeepEquals, i)

	p.client = errorReplyImpl{}
	hostInfo, err = p.GetHostInfo()
	c.Assert(err, ErrorMatches, "fake error")
}
