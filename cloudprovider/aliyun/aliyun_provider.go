package aliyun

import (
	"os"
	"time"

	"github.com/denverdino/aliyungo/metadata"
	"github.com/rancher/agent/cloudprovider"
	"github.com/rancher/agent/core/hostInfo"
)

const (
	aliyunTag = "aliyun"
)

type Provider struct {
	client      metadataClient
	expireTime  time.Duration
	interval    time.Duration
	initialized bool
}

type metadataClient interface {
	Region() (string, error)
	Zone() (string, error)
}

type metadataClientImpl struct {
	client *metadata.MetaData
}

func init() {
	cloudprovider.AddCloudProvider(aliyunTag, &Provider{
		expireTime: time.Minute * 5,
		interval:   time.Second * 10,
	})
}

func (m metadataClientImpl) Region() (string, error) {
	return m.client.Region()
}

func (m metadataClientImpl) Zone() (string, error) {
	return m.client.Zone()
}

func (p *Provider) Init() error {
	client := metadataClientImpl{metadata.NewMetaData(nil)}
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

		zone, err := p.client.Zone()
		if err != nil {
			continue
		}

		region, err := p.client.Region()
		if err != nil {
			continue
		}
		i := hostInfo.Info{}
		i.Labels = map[string]string{}
		i.Labels[cloudprovider.RegionLabel] = region
		i.Labels[cloudprovider.AvailabilityZoneLabel] = zone
		i.Labels[cloudprovider.CloudProviderLabel] = aliyunTag
		if err = cloudprovider.WriteHostInfo(i); err != nil {
			continue
		}
	}
	if _, err := os.Stat(cloudprovider.InfoPath); err == nil {
		success = true
	}
	return success
}
