package hostInfo

import (
	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"github.com/rancher/agent/model"
)

type Collector interface {
	GetData() (map[string]interface{}, error)
	KeyName() string
	GetLabels(string) (map[string]string, error)
}

func CollectData(collectors []Collector) map[string]interface{} {
	data := map[string]interface{}{}
	for _, collector := range collectors {
		collectedData, err := collector.GetData()
		if err != nil {
			logrus.Warnf("Failed to collect data from collector %v. Error msg: %v", collector.KeyName(), err.Error())
		}
		data[collector.KeyName()] = collectedData
	}
	return data
}

func HostLabels(prefix string, collectors []Collector) (map[string]string, error) {
	labels := map[string]string{}
	for _, collector := range collectors {
		lmap, err := collector.GetLabels(prefix)
		if err != nil {
			return map[string]string{}, errors.WithStack(err)
		}
		for key, value := range lmap {
			labels[key] = value
		}
	}
	return labels, nil
}

func GetDefaultDisk(infoData model.InfoData) (string, error) {
	collector := IopsCollector{}
	return collector.getDefaultDisk()
}
