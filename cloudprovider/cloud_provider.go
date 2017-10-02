package cloudprovider

import (
	"encoding/json"
	"os"
	"path"
	"time"

	"github.com/Sirupsen/logrus"

	"github.com/rancher/agent/utils"
	"github.com/rancher/agent/host_info"
)

const (
	CloudProviderLabel    = "io.rancher.host.provider"
	RegionLabel           = "io.rancher.host.region"
	AvailabilityZoneLabel = "io.rancher.host.zone"
	infoFile              = "info.json"
	tempFile              = "temp.json"
)

var (
	InfoPath = path.Join(utils.StateDir(), infoFile)
	TempPath = path.Join(utils.StateDir(), tempFile)
)

var (
	providers = make(map[string]Provider)
)

type Provider interface {
	Init() error
	GetHostInfo() (*hostInfo.Info, error)
	RetryCount() int
	Interval() time.Duration
	Name() string
}

func AddCloudProvider(name string, provider Provider) {
	if _, exists := providers[name]; exists {
		logrus.Fatalf("Provider '%s' tried to register twice", name)
	}
	providers[name] = provider
}

func GetCloudProviderInfo() {
	for name, provider := range providers {
		if err := provider.Init(); err != nil {
			logrus.Fatalf("Provider '%s' initial failed, err: '%s'", name, err)
			continue
		}

		go func(p Provider) {
			for i := 0; ; {
				i++
				if IsHostStateReady() {
					return
				}
				hostInfo, err := p.GetHostInfo()
				if err == nil {
					if err = WriteHostInfo(hostInfo); err == nil {
						logrus.Infof("success get %s host info", p.Name())
						return
					}
				}

				if i >= p.RetryCount() {
					logrus.Errorf("checking %s cloud provider error after %d attempts, last error: %s", p.Name(), i, err)
					return
				}

				logrus.WithFields(logrus.Fields{
					"count": i,
					"error": err,
				}).Error("retry check cloud provider")

				time.Sleep(p.Interval())
			}
		}(provider)
	}
	return
}

func WriteHostInfo(i *hostInfo.Info) error {
	bytes, err := json.Marshal(i)
	if err != nil {
		return err
	}
	file, err := os.Create(TempPath)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.Write(bytes); err != nil {
		return err
	}

	return os.Rename(TempPath, InfoPath)
}

func IsHostStateReady() bool {
	if _, err := os.Stat(InfoPath); err != nil {
		logrus.Warn(err)
		return false
	}
	return true
}
