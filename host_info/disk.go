package hostInfo

import (
	"github.com/pkg/errors"
)

type DiskCollector struct {
	Unit                uint64
	DockerStorageDriver string
}

func (d DiskCollector) GetData() (map[string]interface{}, error) {
	data := map[string]interface{}{
		"fileSystems":               map[string]interface{}{},
		"mountPoints":               map[string]interface{}{},
		"dockerStorageDriverStatus": map[string]interface{}{},
		"dockerStorageDriver":       DockerData.Info.Driver,
	}

	mfs, err := d.getMachineFilesystems()
	if err != nil {
		return data, errors.Wrap(err, "failed get filesystem info")
	}
	for key, value := range mfs {
		data["fileSystems"].(map[string]interface{})[key] = value
	}
	mp, err := d.getMountPoints()
	if err != nil {
		return data, errors.Wrap(err, "failed get mountpoint info")
	}
	for key, value := range mp {
		data["mountPoints"].(map[string]interface{})[key] = value
	}

	return data, nil
}

func (d DiskCollector) KeyName() string {
	return "diskInfo"
}

func (d DiskCollector) GetLabels(prefix string) (map[string]string, error) {
	return map[string]string{}, nil
}
