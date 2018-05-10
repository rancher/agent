package cloudprovider

import (
	"encoding/json"
	"os"
	"path"
	"time"

	"github.com/leodotcloud/log"
	"github.com/rancher/agent/core/hostinfo"
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
	GetHostInfo() (*hostinfo.Info, error)
	RetryCount() int
	Interval() time.Duration
	Name() string
}

func AddCloudProvider(name string, provider Provider) {
	if _, exists := providers[name]; exists {
		log.Fatalf("Provider '%s' tried to register twice", name)
	}
	providers[name] = provider
}

func GetCloudProviderInfo() {
	for name, provider := range providers {
		if err := provider.Init(); err != nil {
			log.Fatalf("Provider '%s' initial failed, err: '%s'", name, err)
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
						log.Infof("success get %s host info", p.Name())
						return
					}
				}

				if i >= p.RetryCount() {
					log.Infof("checking %s cloud provider error after %d attempts with no response, skipping now", p.Name(), i)
					return
				}

				time.Sleep(p.Interval())
			}
		}(provider)
	}
	return
}

func WriteHostInfo(i *hostinfo.Info) error {
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
		return false
	}
	return true
}
