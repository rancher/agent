package aliyun

import (
	"time"

	"github.com/denverdino/aliyungo/metadata"

	"github.com/rancher/agent/cloudprovider"
	"github.com/rancher/agent/host_info"
)

const (
	aliyunTag = "aliyun"
)

type Provider struct {
	client     metadataClient
	interval   time.Duration
	expireTime time.Duration
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
		expireTime: time.Minute * 3,
		interval:   time.Second * 5,
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
	return nil
}

func (p *Provider) Name() string {
	return aliyunTag
}

func (p *Provider) GetHostInfo() (i *hostInfo.Info, err error) {
	zone, err := p.client.Zone()
	if err != nil {
		return
	}
	region, err := p.client.Region()
	if err != nil {
		return
	}
	i = &hostInfo.Info{}
	i.Labels = map[string]string{}
	i.Labels[cloudprovider.RegionLabel] = region
	i.Labels[cloudprovider.AvailabilityZoneLabel] = zone
	i.Labels[cloudprovider.CloudProviderLabel] = aliyunTag
	return
}

func (p *Provider) ExpireTime() time.Duration {
	return p.expireTime
}

func (p *Provider) Interval() time.Duration {
	return p.interval
}
