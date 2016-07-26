package hostInfo

import (
	"bufio"
	"github.com/Sirupsen/logrus"
	"math"
	"os"
	"runtime"
	"strconv"
	"strings"
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

type MemoryCollector struct {
	unit float64
}

func (m MemoryCollector) getMemInfoData() []string {
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
	logrus.Infof("memory info %v", data)
	return data
}

func (m MemoryCollector) parseLinuxMemInfo() map[string]interface{} {
	data := map[string]interface{}{}
	memData := m.getMemInfoData()
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

func (m MemoryCollector) GetData() map[string]interface{} {
	if runtime.GOOS == "linux" {
		return m.parseLinuxMemInfo()
	}
	return map[string]interface{}{}
}

func (m MemoryCollector) KeyName() string {
	return "memoryInfo"
}

func (m MemoryCollector) GetLabels(prefix string) map[string]string {
	return map[string]string{}
}
