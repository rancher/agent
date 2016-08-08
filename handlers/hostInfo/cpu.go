package hostInfo

import (
	"bufio"
	"fmt"
	"github.com/Sirupsen/logrus"
	"math"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

type CPUCollector struct {
	cadvisor CadvisorAPIClient
}

func (c CPUCollector) getCPUInfoData() []string {
	if runtime.GOOS == "linux" {
		file, err := os.Open("/proc/cpuinfo")
		defer file.Close()
		data := []string{}
		if err != nil {
			logrus.Error(err)
		} else {
			scanner := bufio.NewScanner(file)
			scanner.Split(bufio.ScanLines)
			for scanner.Scan() {
				data = append(data, scanner.Text())
			}
		}
		return data
	}
	return []string{}
}

func (c CPUCollector) getLinuxCPUInfo() map[string]interface{} {
	data := map[string]interface{}{}

	procs := []string{}
	if runtime.GOOS == "linux" {
		fileData := c.getCPUInfoData()
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
	}

	return data
}

func (c CPUCollector) getCPUPercentage() map[string]interface{} {
	data := map[string]interface{}{}
	cpuCoresPercentages := []string{}

	stats := c.cadvisor.GetStats()

	if len(stats) >= 2 {
		statLatest := stats[len(stats)-1].(map[string]interface{})
		statPrev := stats[len(stats)-2].(map[string]interface{})

		timeDiff := c.cadvisor.TimestampDiff(InterfaceToString(statLatest["timestamp"]), InterfaceToString(statPrev["timestamp"].(string)))
		latestUsage, _ := getFieldsIfExist(statLatest, "cpu", "usage", "per_cpu_usage")
		prevUsage, _ := getFieldsIfExist(statPrev, "cpu", "usage", "per_cpu_usage")
		for i, cu := range InterfaceToArray(latestUsage) {
			coreUsage := InterfaceToString(cu)
			core, _ := strconv.ParseFloat(coreUsage, 64)
			pu := InterfaceToString(InterfaceToArray(prevUsage)[i])
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

func (c CPUCollector) getLoadAverage() map[string]interface{} {
	// TODO mock not implemented
	return map[string]interface{}{
		"loadAvg": getLoadAverage(),
	}
}

func (c CPUCollector) GetData() map[string]interface{} {
	data := map[string]interface{}{}

	if runtime.GOOS == "linux" {
		for key, value := range c.getLinuxCPUInfo() {
			data[key] = value
		}
		for key, value := range c.getCPUPercentage() {
			data[key] = value
		}
		for key, value := range c.getLoadAverage() {
			data[key] = value
		}
	}
	return data
}

func (c CPUCollector) KeyName() string {
	return "cpuInfo"
}

func (c CPUCollector) GetLabels(prefix string) map[string]string {
	if _, err := os.Stat("/dev/kvm"); err == nil {
		return map[string]string{
			fmt.Sprintf("%s.%s", prefix, "kvm"): "true",
		}
	}
	return map[string]string{}
}
