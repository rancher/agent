package hostInfo

import (
	"bufio"
	"github.com/Sirupsen/logrus"
	"math"
	"os"
	"strconv"
	"strings"
	"os/exec"
)

var KeyMap = map[string]string{
	"memtotal":     "memTotal",
	"memfree":      "memFree",
	"memavailable": "memAvailable",
	"buffers":      "buffers",
	"cached":       "cached",
	"swapcached":   "swapCached",
	"active":       "active",
	"inactive":     "inactive",
	"swaptotal":    "swapTotal",
	"swapfree":     "swapFree",
}

type MemoryInfoGetter interface {
	GetMemInfoData() []string
}

type MemoryCollector struct {
	dataGetter MemoryInfoGetter
	unit       float64
	GOOS       string
}

type MemoryDataGetter struct{}

func (m MemoryDataGetter) GetMemInfoData() []string {
	file, err := os.Open("/proc/meminfo")
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

func (m MemoryCollector) parseLinuxMemInfo() map[string]interface{} {
	data := map[string]interface{}{}
	memData := m.dataGetter.GetMemInfoData()
	for _, line := range memData {
		lineList := strings.Split(line, ":")
		keyLower := strings.ToLower(lineList[0])
		possibleMemValue := strings.Split(strings.TrimSpace(lineList[1]), " ")[0]

		if _, ok := KeyMap[keyLower]; ok {
			value, _ := strconv.ParseFloat(possibleMemValue, 64)
			convertMemVal := value / m.unit
			convertMemVal = math.Floor(convertMemVal*1000) / 1000
			data[KeyMap[keyLower]] = convertMemVal
		}
	}
	return data
}

func (m MemoryCollector) parseWindowsMemInfo() map[string]interface{} {
	data := map[string]interface{}{}
	keys := []string{
		"FreePhysicalMemory",
		"FreeVirtualMemory",
		"TotalSwapSpaceSize",
		"TotalVirtualMemorySize",
		"TotalVisibleMemorySize",
	}
	for _, key := range keys {
		value, err := getCommandOutput(key)
		if err != nil {
			logrus.Error(err)
		} else {
			data[key] = value
		}
	}
	return data
}

func getCommandOutput(key string) (string, error) {
	command := exec.Command("PowerShell", "wmic", "os", "get", key)
	output, err := command.Output()
	if err == nil {
		ret := strings.Split(string(output), "\n")[1]
		return ret, nil
	} else {
		return "", err
	}
}

func (m MemoryCollector) GetData() map[string]interface{} {
	if m.GOOS == "linux" {
		return m.parseLinuxMemInfo()
	} else if m.GOOS == "windows" {
		return m.parseWindowsMemInfo()
	}
	return nil
}

func (m MemoryCollector) KeyName() string {
	return "memoryInfo"
}

func (m MemoryCollector) GetLabels(prefix string) map[string]string {
	return map[string]string{}
}
