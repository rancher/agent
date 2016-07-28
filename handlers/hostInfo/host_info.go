package hostInfo

import (
	"fmt"
	"github.com/rancher/agent/handlers/docker"
)

type Collector interface {
	GetData() map[string]interface{}
	KeyName() string
	GetLabels(string) map[string]string
}

var Cadvisor = CadvisorAPIClient{URL: fmt.Sprintf("%v%v:%v/api/%v", "http://", CadvisorIP(), CadvisorPort(),
	"v1.2")}

var Collectors = []Collector{
	CPUCollector{cadvisor: Cadvisor},
	DiskCollector{cadvisor: Cadvisor, client: docker.GetClient(DefaultVersion),
		dockerStorageDriver: getInfo().Driver, unit: 1048576},
	IopsCollector{data: map[string]interface{}{}},
	MemoryCollector{unit: 1024.00},
	OSCollector{client: docker.GetClient(DefaultVersion)},
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
