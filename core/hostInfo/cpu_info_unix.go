// +build linux freebsd solaris openbsd darwin

package hostInfo

import (
	"bufio"
	"github.com/Sirupsen/logrus"
	linuxproc "github.com/c9s/goprocinfo/linux"
	"github.com/pkg/errors"
	"github.com/rancher/agent/utilities/constants"
	"github.com/rancher/agent/utilities/utils"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
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

	stats, err := getStats()
	if err != nil {
		logrus.Error(err)
		return nil
	}
	per1 := []float64{}
	per2 := []float64{}
	for _, cpuStats := range stats[0].CPUStats {
		per1 = append(per1, float64((cpuStats.User+cpuStats.System)/(cpuStats.User+cpuStats.System+cpuStats.Idle)*100))
	}
	for _, cpuStats := range stats[1].CPUStats {
		per2 = append(per1, float64((cpuStats.User+cpuStats.System)/(cpuStats.User+cpuStats.System+cpuStats.Idle)*100.00))
	}
	for i := range per1 {
		diff := per1[i] - per2[i]
		cpuCoresPercentages = append(cpuCoresPercentages, strconv.FormatFloat(diff, 'f', -1, 64))
	}
	data["cpuCoresPercentages"] = cpuCoresPercentages
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

func timestampDiff(timeCurrent, timePrev string) float64 {
	timeCurConv, _ := time.Parse(time.RFC3339, timeCurrent[0:26])
	timePrevConv, _ := time.Parse(time.RFC3339, timePrev[0:26])
	diff := timeCurConv.Sub(timePrevConv)
	return float64(diff)
}

func getStats() ([2]*linuxproc.Stat, error) {
	ret := [2]*linuxproc.Stat{}
	statsPrev, err := linuxproc.ReadStat("/proc/stat")
	if err != nil {
		return ret, err
	}
	time.Sleep(1 * time.Second)
	statsLatest, err := linuxproc.ReadStat("/proc/stat")
	if err != nil {
		return ret, err
	}
	ret[0] = statsLatest
	ret[1] = statsPrev
	return ret, nil
}
