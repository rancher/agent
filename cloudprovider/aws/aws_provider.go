package aws

import (
	"os"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/rancher/agent/cloudprovider"
	"github.com/rancher/agent/core/hostInfo"
)

const (
	awsTag = "aws"
)

type Provider struct {
	client      metadataClient
	expireTime  time.Duration
	interval    time.Duration
	initialized bool
}

type metadataClient interface {
	getInstanceIdentityDocument() (ec2metadata.EC2InstanceIdentityDocument, error)
}

type metadataClientImpl struct {
	client *ec2metadata.EC2Metadata
}

func init() {
	cloudprovider.AddCloudProvider(awsTag, &Provider{
		expireTime: time.Minute * 5,
		interval:   time.Second * 10,
	})
}

func (m metadataClientImpl) getInstanceIdentityDocument() (ec2metadata.EC2InstanceIdentityDocument, error) {
	return m.client.GetInstanceIdentityDocument()
}

func (p *Provider) Init() error {
	s, err := session.NewSession()
	if err != nil {
		logrus.Error(err)
		return err
	}
	client := metadataClientImpl{ec2metadata.New(s)}
	p.client = client
	p.initialized = true
	return nil
}

func (p *Provider) GetCloudProviderInfo() bool {
	if !p.initialized {
		return false
	}
	success := false
	endtime := time.Now().Add(p.expireTime)
	for {
		if time.Now().After(endtime) {
			break
		}
		if _, err := os.Stat(cloudprovider.InfoPath); err == nil {
			break
		}
		time.Sleep(p.interval)

		document, err := p.client.getInstanceIdentityDocument()
		if err != nil {
			continue
		}
		i := hostInfo.Info{}
		i.Labels = map[string]string{}
		i.Labels[cloudprovider.RegionLabel] = document.Region
		i.Labels[cloudprovider.AvailabilityZoneLabel] = document.AvailabilityZone
		i.Labels[cloudprovider.CloudProviderLabel] = awsTag
		if err = cloudprovider.WriteHostInfo(i); err != nil {
			continue
		}
	}
	if _, err := os.Stat(cloudprovider.InfoPath); err == nil {
		success = true
	}
	return success
}
