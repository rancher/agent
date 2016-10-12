// +build linux freebsd solaris openbsd darwin

package hostInfo

import (
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/load"
	"strconv"
	"time"
)

func (c CPUCollector) getCPUInfo() (map[string]interface{}, error) {
	data := map[string]interface{}{}

	cpuInfo, err := cpu.Info()
	if err != nil {
		return map[string]interface{}{}, err
	}
	counts, err := cpu.Counts(true)
	if err != nil {
		return map[string]interface{}{}, err
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
	cpuCoresPercentages := []float64{}

	percents, err := cpu.Percent(time.Second*1, true)
	if err != nil {
		return map[string]interface{}{}, err
	}
	for _, percent := range percents {
		cpuCoresPercentages = append(cpuCoresPercentages, percent)
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
