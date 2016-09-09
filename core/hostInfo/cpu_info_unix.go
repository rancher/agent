// +build linux freebsd solaris openbsd darwin

package hostInfo

import (
	"bufio"
	"github.com/pkg/errors"
	"github.com/rancher/agent/utilities/constants"
	"github.com/rancher/agent/utilities/utils"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
)

func (c CPUCollector) getCPUInfo() (map[string]interface{}, error) {
	data := map[string]interface{}{}

	procs := []string{}
	fileData, err := c.DataGetter.GetCPUInfoData()
	if err != nil {
		return data, errors.Wrap(err, constants.GetCPUInfoError+"failed to get cpu info")
	}
	for _, line := range fileData {
		parts := strings.Split(line, ":")
		if strings.TrimSpace(parts[0]) == "model name" {
			procs = append(procs, strings.TrimSpace(parts[1]))
			pattern := "([0-9\\.]+)\\s?GHz"
			freq := regexp.MustCompile(pattern).FindString(parts[1])
			if freq != "" {
				ghz := strings.TrimSpace(freq[:len(freq)-3])
				if ghz != "" {
					mhz, _ := strconv.ParseFloat(ghz, 64)
					data["mhz"] = mhz * 1000
				}
			}
		}
		if _, ok := data["mhz"]; !ok {
			if strings.TrimSpace(parts[0]) == "cpu MHz" {
				mhz, _ := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
				data["mhz"] = mhz
			}
		}
	}
	data["modelName"] = procs[0]
	data["count"] = len(procs)

	return data, nil
}

func (c CPUDataGetter) GetCPUInfoData() ([]string, error) {
	file, err := os.Open("/proc/cpuinfo")
	defer file.Close()
	data := []string{}
	if err != nil {
		return data, errors.Wrap(err, constants.GetCPUInfoDataError+"failed to open cpuinfo file")
	}
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		data = append(data, scanner.Text())
	}

	return data, nil
}

func (c CPUCollector) getCPUPercentage() map[string]interface{} {
	data := map[string]interface{}{}
	cpuCoresPercentages := []string{}

	stats := c.Cadvisor.GetStats()

	if len(stats) >= 2 {
		statLatest := stats[len(stats)-1].(map[string]interface{})
		statPrev := stats[len(stats)-2].(map[string]interface{})

		timeDiff := c.Cadvisor.TimestampDiff(utils.InterfaceToString(statLatest["timestamp"]), utils.InterfaceToString(statPrev["timestamp"].(string)))
		latestUsage, _ := utils.GetFieldsIfExist(statLatest, "cpu", "usage", "per_cpu_usage")
		prevUsage, _ := utils.GetFieldsIfExist(statPrev, "cpu", "usage", "per_cpu_usage")
		for i, cu := range utils.InterfaceToArray(latestUsage) {
			coreUsage := utils.InterfaceToString(cu)
			core, _ := strconv.ParseFloat(coreUsage, 64)
			pu := utils.InterfaceToString(utils.InterfaceToArray(prevUsage)[i])
			prev, _ := strconv.ParseFloat(pu, 64)
			cpuUsage := core - prev
			percentage := (cpuUsage / float64(timeDiff)) * 100
			percentage = percentage * 1000 // round to 3
			if percentage > 100000 {
				percentage = 100
			} else {
				percentage = math.Floor(percentage) / 1000
			}
			cpuCoresPercentages = append(cpuCoresPercentages, strconv.FormatFloat(percentage, 'f', -1, 64))
		}
		data["cpuCoresPercentages"] = cpuCoresPercentages
	}
	return data
}

func (c CPUDataGetter) GetCPULoadAverage() (map[string]interface{}, error) {
	loadAvg, err := utils.GetLoadAverage()
	if err != nil {
		return map[string]interface{}{}, err
	}
	return map[string]interface{}{
		"loadAvg": loadAvg,
	}, nil
}
