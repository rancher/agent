package hostInfo

import "github.com/rancher/agent/model"

func (d DiskCollector) getMachineFilesystemsCadvisor(infoData model.InfoData) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (d DiskCollector) getMountPointsCadvisor() map[string]interface{} {
	return map[string]interface{}{}
}

func (d DiskDataGetter) GetDockerStorageInfo(infoData model.InfoData) map[string]interface{} {
	return map[string]interface{}{}
}
