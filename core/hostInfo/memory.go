package hostInfo

import (
	"bufio"
	"github.com/pkg/errors"
	"github.com/rancher/agent/utilities/constants"
	"os"
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
	GetMemInfoData() ([]string, error)
}

type MemoryCollector struct {
	DataGetter MemoryInfoGetter
	Unit       float64
	GOOS       string
}

type MemoryDataGetter struct{}

func (m MemoryDataGetter) GetMemInfoData() ([]string, error) {
	file, err := os.Open("/proc/meminfo")
	defer file.Close()
	data := []string{}
	if err != nil {
		return data, errors.Wrap(err, constants.GetMemInfoDataError+"failed to read meminfo file")
	}
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		data = append(data, scanner.Text())
	}
	return data, nil
}

func (m MemoryCollector) GetData() (map[string]interface{}, error) {
	return m.parseMemInfo()
}

func (m MemoryCollector) KeyName() string {
	return "memoryInfo"
}

func (m MemoryCollector) GetLabels(prefix string) (map[string]string, error) {
	return map[string]string{}, nil
}
