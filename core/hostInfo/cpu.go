package hostInfo

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/rancher/agent/utilities/constants"
	"os"
)

type CPUInfoGetter interface {
	GetCPUInfoData() ([]string, error)
	GetCPULoadAverage() (map[string]interface{}, error)
}

type CPUCollector struct {
	Cadvisor   CadvisorAPIClient
	DataGetter CPUInfoGetter
	GOOS       string
}

type CPUDataGetter struct{}

func (c CPUCollector) GetData() (map[string]interface{}, error) {
	data := map[string]interface{}{}

	cInfo, err := c.getCPUInfo()
	if err != nil {
		return data, errors.Wrap(err, constants.CPUGetDataError+"failed to get cpu info")
	}
	for key, value := range c.getCPUPercentage() {
		data[key] = value
	}
	for key, value := range cInfo {
		data[key] = value
	}
	loadAvg, err := c.DataGetter.GetCPULoadAverage()
	if err != nil {
		return map[string]interface{}{}, errors.Wrap(err, constants.CPUGetDataError+"failed to get cpu load average")
	}
	for key, value := range loadAvg {
		data[key] = value
	}
	return data, nil
}

func (c CPUCollector) KeyName() string {
	return "cpuInfo"
}

func (c CPUCollector) GetLabels(prefix string) (map[string]string, error) {
	if _, err := os.Stat("/dev/kvm"); err == nil {
		return map[string]string{
			fmt.Sprintf("%s.%s", prefix, "kvm"): "true",
		}, nil
	}
	return map[string]string{}, nil
}
