package aliyun

import (
	"encoding/json"
	"os"
	"path"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/denverdino/aliyungo/metadata"
	"github.com/rancher/agent/core/hostInfo"
	"github.com/rancher/agent/utilities/config"
)

const (
	cloudProviderLabel    = "io.rancher.host.provider"
	regionLabel           = "io.rancher.host.region"
	availabilityZoneLabel = "io.rancher.host.zone"
	infoFile              = "info.json"
	tempFile              = "temp.json"
	aliyunTag             = "aliyun"
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

func (m metadataClientImpl) Region() (string, error) {
	return m.client.Region()
}

func (m metadataClientImpl) Zone() (string, error) {
	return m.client.Zone()
}

func NewProvider() Provider {
	prod := Provider{}
	client := metadata.NewMetaData(nil)
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
		i.Labels[regionLabel] = region
		i.Labels[availabilityZoneLabel] = zone
		i.Labels[cloudProviderLabel] = aliyunTag
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
