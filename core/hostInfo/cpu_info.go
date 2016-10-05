// +build linux freebsd solaris openbsd darwin

package hostInfo

import (
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/load"
)

func (c CPUCollector) getCPUInfo() (map[string]interface{}, error) {
	data := map[string]interface{}{}

	cpuInfo, err := cpu.Info()
	if err != nil {
		return map[string]interface{}{}, errors.WithStack(err)
	}
	counts, err := cpu.Counts(true)
	if err != nil {
		return map[string]interface{}{}, errors.WithStack(err)
	}
	data["count"] = counts
	if len(cpuInfo) > 0 {
		data["modelName"] = cpuInfo[0].ModelName
		data["mhz"] = cpuInfo[0].Mhz
	}

	return data, nil
}

func (c CPUCollector) getCPUPercentage() (map[string]interface{}, error) {
	data := map[string]interface{}{}
	cpuCoresPercentages := []string{}

	percents, err := cpu.Percent(time.Second*1, true)
	if err != nil {
		return map[string]interface{}{}, errors.WithStack(err)
	}
	for _, percent := range percents {
		cpuCoresPercentages = append(cpuCoresPercentages, strconv.FormatFloat(percent, 'f', -1, 64))
	}
	data["cpuCoresPercentages"] = cpuCoresPercentages
	return data, nil
}

func (c CPUCollector) getCPULoadAverage() map[string]interface{} {
	loadData, err := load.Avg()
	if err != nil {
		return map[string]interface{}{}
	}
	loadAvg := []string{}
	loadAvg = append(loadAvg, strconv.FormatFloat(loadData.Load1, 'f', -1, 64))
	loadAvg = append(loadAvg, strconv.FormatFloat(loadData.Load5, 'f', -1, 64))
	loadAvg = append(loadAvg, strconv.FormatFloat(loadData.Load15, 'f', -1, 64))
	return map[string]interface{}{
		"loadAvg": loadAvg,
	}
}
