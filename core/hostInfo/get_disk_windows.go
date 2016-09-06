package hostInfo

func (d DiskCollector) getMachineFilesystemsCadvisor() (map[string]interface{}, error) {}

func (d DiskCollector) getMountPointsCadvisor() map[string]interface{} {}

func (d DiskDataGetter) GetDockerStorageInfo() map[string]interface{} {}
