package hostInfo

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
)

type CPUCollector struct {
}

func (c CPUCollector) GetData() (map[string]interface{}, error) {
	data := map[string]interface{}{}

	cInfo, err := c.getCPUInfo()
	if err != nil {
		return data, errors.WithStack(err)
	}
	percent, err := c.getCPUPercentage()
	if err != nil {
		return data, errors.WithStack(err)
	}
	for key, value := range percent {
		data[key] = value
	}
	for key, value := range cInfo {
		data[key] = value
	}
	for key, value := range c.getCPULoadAverage() {
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
