package hostInfo

import (
	"os/exec"
	"strings"
	"regexp"
	"strconv"
	"runtime"
	"github.com/pkg/errors"
	"github.com/rancher/agent/utilities/constants"
)

func (c CPUCollector) getCPUInfo() (map[string]interface{}, error) {
	data := map[string]interface{}{}
	command := exec.Command("PowerShell", "wmic", "cpu", "get", "Name")
	output, err := command.Output()
	if err != nil {
		return data, errors.Wrap(err, constants.GetCPUInfoError)
	}
	ret := strings.Split(string(output), "\n")[1]
	data["modelName"] = ret
	pattern := "([0-9\\.]+)\\s?GHz"
	freq := regexp.MustCompile(pattern).FindString(ret)
	if freq != "" {
		ghz := strings.TrimSpace(freq[:len(freq)-3])
		if ghz != "" {
			mhz, _ := strconv.ParseFloat(ghz, 64)
			data["mhz"] = mhz * 1000
		}
	}
	data["count"] = runtime.NumCPU()
	return data, nil
}

func (c CPUCollector) getCPUPercentage() map[string]interface{} {
	return map[string]interface{}{}
}

func (c CPUDataGetter) GetCPULoadAverage() (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (c CPUDataGetter) GetCPUInfoData() ([]string, error) {
	return []string{}, nil
}
