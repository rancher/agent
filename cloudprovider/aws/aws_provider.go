package aws

import (
	"time"

	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"

	"github.com/rancher/agent/cloudprovider"
	"github.com/rancher/agent/host_info"
)

const (
	awsTag = "aws"
)

type Provider struct {
	client     metadataClient
	interval   time.Duration
	expireTime time.Duration
}

type metadataClient interface {
	getInstanceIdentityDocument() (ec2metadata.EC2InstanceIdentityDocument, error)
}

type metadataClientImpl struct {
	client *ec2metadata.EC2Metadata
}

func init() {
	cloudprovider.AddCloudProvider(awsTag, &Provider{
		expireTime: time.Minute * 3,
		interval:   time.Microsecond * 500,
	})
}

func (m metadataClientImpl) getInstanceIdentityDocument() (ec2metadata.EC2InstanceIdentityDocument, error) {
	return m.client.GetInstanceIdentityDocument()
}

func (p *Provider) Init() error {
	s, err := session.NewSession()
	if err != nil {
		return err
	}
	client := metadataClientImpl{ec2metadata.New(s)}
	p.client = client
	return nil
}

func (p *Provider) Name() string {
	return awsTag
}

func (p *Provider) GetHostInfo() (i *hostInfo.Info, err error) {
	document, err := p.client.getInstanceIdentityDocument()
	if err != nil {
		return
	}
	i = &hostInfo.Info{}
	i.Labels = map[string]string{}
	i.Labels[cloudprovider.RegionLabel] = document.Region
	i.Labels[cloudprovider.AvailabilityZoneLabel] = document.AvailabilityZone
	i.Labels[cloudprovider.CloudProviderLabel] = awsTag
	return
}

func (p *Provider) ExpireTime() time.Duration {
	return p.expireTime
}

func (p *Provider) Interval() time.Duration {
	return p.interval
}
