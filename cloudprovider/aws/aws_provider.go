package aws

import (
	"time"

	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"

	"github.com/rancher/agent/cloudprovider"
	"github.com/rancher/agent/core/hostInfo"
)

const (
	AwsTag                     = "aws"
	awsInternalHostnameAPIPath = "local-hostname"
)

type Provider struct {
	client     metadataClient
	interval   time.Duration
	retryCount int
}

type metadataClient interface {
	getInstanceIdentityDocument() (ec2metadata.EC2InstanceIdentityDocument, error)
	getMetadata(string) (string, error)
}

type metadataClientImpl struct {
	client *ec2metadata.EC2Metadata
}

func init() {
	cloudprovider.AddCloudProvider(AwsTag, &Provider{
		retryCount: 6,
		interval:   time.Second * 30,
	})
}

func (m metadataClientImpl) getInstanceIdentityDocument() (ec2metadata.EC2InstanceIdentityDocument, error) {
	return m.client.GetInstanceIdentityDocument()
}

func (m metadataClientImpl) getMetadata(path string) (string, error) {
	return m.client.GetMetadata(path)
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
	return AwsTag
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
	i.Labels[cloudprovider.CloudProviderLabel] = AwsTag
	return
}

func (p *Provider) RetryCount() int {
	return p.retryCount
}

func (p *Provider) Interval() time.Duration {
	return p.interval
}

func (p *Provider) GetAWSLocalHostname() (string, error) {
	localHostname, err := p.client.getMetadata(awsInternalHostnameAPIPath)
	if err != nil {
		return "", err
	}
	return localHostname, nil
}
