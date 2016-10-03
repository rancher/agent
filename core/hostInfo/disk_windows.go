package hostInfo

import "github.com/rancher/agent/model"

func (d DiskCollector) getMachineFilesystems(infoData model.InfoData) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (d DiskCollector) getMountPoints() map[string]interface{} {
	return map[string]interface{}{}
}

func (d DiskDataGetter) GetDockerStorageInfo(infoData model.InfoData) map[string]interface{} {
	return map[string]interface{}{}
}
