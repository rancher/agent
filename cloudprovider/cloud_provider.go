package cloudprovider

import (
	"encoding/json"
	"os"
	"path"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/agent/core/hostInfo"
	"github.com/rancher/agent/utilities/config"
)

const (
	CloudProviderLabel    = "io.rancher.host.provider"
	RegionLabel           = "io.rancher.host.region"
	AvailabilityZoneLabel = "io.rancher.host.zone"
	infoFile              = "info.json"
	tempFile              = "temp.json"
)

var (
	InfoPath = path.Join(config.StateDir(), infoFile)
	TempPath = path.Join(config.StateDir(), tempFile)
)

var (
	providers = make(map[string]Provider)
)

type Provider interface {
	Init() error
	GetCloudProviderInfo() bool
}

func AddCloudProvider(name string, provider Provider) {
	if _, exists := providers[name]; exists {
		logrus.Fatalf("Provider '%s' tried to register twice", name)
	}
	logrus.Infof("Provider '%s' adding", name)
	providers[name] = provider
}

func GetCloudProviderInfo() {
	for name, provider := range providers {
		if err := provider.Init(); err != nil {
			logrus.Fatalf("Provider '%s' initial failed, err: '%s'", name, err)
			continue
		}
		go provider.GetCloudProviderInfo()
	}
}

func WriteHostInfo(i hostInfo.Info) error {
	bytes, err := json.Marshal(i)
	if err != nil {
		logrus.Error(err)
		return err
	}
	file, err := os.Create(TempPath)
	if err != nil {
		logrus.Error(err)
		return err
	}
	defer file.Close()
	if _, err = file.Write(bytes); err != nil {
		logrus.Error(err)
		return err
	}

	if err = os.Rename(TempPath, InfoPath); err != nil {
		logrus.Error(err)
		return err
	}
	return nil
}
