package aws

import (
	"encoding/json"
	"os"
	"path"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/rancher/agent/core/hostInfo"
	"github.com/rancher/agent/utils/config"
)

const (
	cloudProviderLabel    = "io.rancher.host.provider"
	regionLabel           = "io.rancher.host.region"
	availabilityZoneLabel = "io.rancher.host.zone"
	infoFile              = "info.json"
	tempFile              = "temp.json"
	awsTag                = "aws"
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

func (m metadataClientImpl) getInstanceIdentityDocument() (ec2metadata.EC2InstanceIdentityDocument, error) {
	return m.client.GetInstanceIdentityDocument()
}

func NewProvider() Provider {
	prod := Provider{}
	s, err := session.NewSession()
	if err != nil {
		logrus.Error(err)
		return Provider{}
	}
	client := metadataClientImpl{ec2metadata.New(s)}
	prod.client = client
	prod.expireTime = time.Minute * 5
	prod.interval = time.Second * 10
	prod.initialized = true
	return prod
}

func (p Provider) GetCloudProviderInfo() bool {
	if !p.initialized {
		return false
	}
	success := false
	infoPath := path.Join(config.StateDir(), infoFile)
	tempPath := path.Join(config.StateDir(), tempFile)
	endtime := time.Now().Add(p.expireTime)
	for {
		if time.Now().After(endtime) {
			break
		}
		if _, err := os.Stat(infoPath); err == nil {
			break
		}
		time.Sleep(p.interval)

		document, err := p.client.getInstanceIdentityDocument()
		if err != nil {
			continue
		}
		i := hostInfo.Info{}
		i.Labels = map[string]string{}
		i.Labels[regionLabel] = document.Region
		i.Labels[availabilityZoneLabel] = document.AvailabilityZone
		i.Labels[cloudProviderLabel] = awsTag
		bytes, err := json.Marshal(i)
		if err != nil {
			logrus.Error(err)
			continue
		}
		file, err := os.Create(tempPath)
		if err != nil {
			logrus.Error(err)
			continue
		}
		defer file.Close()
		_, err = file.Write(bytes)
		if err != nil {
			logrus.Error(err)
			continue
		}
		err = os.Rename(tempPath, infoPath)
		if err != nil {
			logrus.Error(err)
			continue
		}
	}
	if _, err := os.Stat(infoPath); err == nil {
		success = true
	}
	return success
}
