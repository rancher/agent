package hostInfo

import (
	"github.com/shirou/gopsutil/cpu"
)

func (c *CPUCollector) getCPUInfo() (map[string]interface{}, error) {
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
