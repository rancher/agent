package aws

import (
	"time"

	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"

	"github.com/rancher/agent/cloudprovider"
	"github.com/rancher/agent/core/hostinfo"
)

const (
	awsTag = "aws"
)

type Provider struct {
	client     metadataClient
	interval   time.Duration
	retryCount int
}

type metadataClient interface {
	getInstanceIdentityDocument() (ec2metadata.EC2InstanceIdentityDocument, error)
}

type metadataClientImpl struct {
	client *ec2metadata.EC2Metadata
}

func init() {
	cloudprovider.AddCloudProvider(awsTag, &Provider{
		retryCount: 6,
		interval:   time.Second * 30,
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

func (p *Provider) GetHostInfo() (i *hostinfo.Info, err error) {
	document, err := p.client.getInstanceIdentityDocument()
	if err != nil {
		return
	}
	i = &hostinfo.Info{}
	i.Labels = map[string]string{}
	i.Labels[cloudprovider.RegionLabel] = document.Region
	i.Labels[cloudprovider.AvailabilityZoneLabel] = document.AvailabilityZone
	i.Labels[cloudprovider.CloudProviderLabel] = awsTag
	return
}

func (p *Provider) RetryCount() int {
	return p.retryCount
}

func (p *Provider) Interval() time.Duration {
	return p.interval
}
