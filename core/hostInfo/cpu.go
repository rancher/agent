package hostInfo

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/rancher/agent/utilities/config"
	"github.com/rancher/agent/utilities/constants"
	"os"
	"time"
)

type CPUCollector struct {
	cacheData map[string]interface{}
	lastRead  time.Time
}

func (c *CPUCollector) GetData() (map[string]interface{}, error) {
	if c.cacheData != nil && time.Now().Before(c.lastRead.Add(time.Minute*time.Duration(config.RefreshInterval()))) {
		return c.cacheData, nil
	}

	data := map[string]interface{}{}

	cInfo, err := c.getCPUInfo()
	if err != nil {
		return data, errors.Wrap(err, constants.CPUGetDataError+"failed to get cpu info")
	}
	for key, value := range cInfo {
		data[key] = value
	}
	c.cacheData = data
	c.lastRead = time.Now()
	return data, nil
}

func (c *CPUCollector) KeyName() string {
	return "cpuInfo"
}

func (c *CPUCollector) GetLabels(prefix string) (map[string]string, error) {
	if _, err := os.Stat("/dev/kvm"); err == nil {
		return map[string]string{
			fmt.Sprintf("%s.%s", prefix, "kvm"): "true",
		}, nil
	}
	return map[string]string{}, nil
}
