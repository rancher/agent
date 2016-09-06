package hostInfo

import (
	"github.com/pkg/errors"
	"github.com/rancher/agent/utilities/constants"
	"regexp"
	"strconv"
	"strings"
	"os/exec"
)

func (m MemoryCollector) parseMemInfo() (map[string]interface{}, error) {
	data := map[string]interface{}{}
	keys := map[string]string{
		"memFree":  "FreePhysicalMemory",
		"memTotal": "TotalVisibleMemorySize",
	}
	for k, v := range keys {
		value, err := getCommandOutput(v)
		if err != nil {
			return map[string]interface{}{}, errors.Wrap(err, constants.ParseMemInfoError)
		}
		pattern := "([0-9]+)"
		possibleMemValue := regexp.MustCompile(pattern).FindString(value)
		memValue, _ := strconv.ParseFloat(possibleMemValue, 64)
		data[k] = memValue / 1024
	}
	return data, nil
}

func getCommandOutput(key string) (string, error) {
	command := exec.Command("PowerShell", "wmic", "os", "get", key)
	output, err := command.Output()
	if err == nil {
		ret := strings.Split(string(output), "\n")[1]
		return ret, nil
	}
	return "", err
}
