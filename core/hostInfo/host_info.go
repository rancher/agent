package hostInfo

import (
	"fmt"
	"github.com/rancher/agent/utilities/config"
	"github.com/rancher/agent/utilities/utils"
	"runtime"
)

type Collector interface {
	GetData() map[string]interface{}
	KeyName() string
	GetLabels(string) map[string]string
}

var Cadvisor = CadvisorAPIClient{
	dataGetter: CadvisorDataGetter{
		URL: fmt.Sprintf("%v%v:%v/api/%v", "http://", config.CadvisorIP(), config.CadvisorPort(), "v1.2"),
	},
}

var Collectors = []Collector{
	CPUCollector{
		cadvisor:   Cadvisor,
		dataGetter: CPUDataGetter{},
		GOOS:       runtime.GOOS,
	},
	DiskCollector{
		cadvisor:            Cadvisor,
		dockerStorageDriver: utils.GetInfoDriver(),
		unit:                1048576,
		dataGetter:          DiskDataGetter{},
	},
	IopsCollector{
		GOOS: runtime.GOOS,
	},
	MemoryCollector{
		unit:       1024.00,
		dataGetter: MemoryDataGetter{},
		GOOS:       runtime.GOOS,
	},
	OSCollector{
		dataGetter: OSDataGetter{},
		GOOS:       runtime.GOOS,
	},
}

func CollectData() map[string]interface{} {
	data := map[string]interface{}{}
	for _, collector := range Collectors {
		data[collector.KeyName()] = collector.GetData()
	}
	return data
}

func HostLabels(prefix string) map[string]string {
	labels := map[string]string{}
	for _, collector := range Collectors {
		for key, value := range collector.GetLabels(prefix) {
			labels[key] = value
		}
	}
	return labels
}

func GetDefaultDisk() string {
	return Collectors[2].(IopsCollector).getDefaultDisk()
}
