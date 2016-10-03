package hostInfo

import (
	info "github.com/google/cadvisor/info/v1"
	"github.com/pkg/errors"
	"github.com/rancher/agent/model"
	"github.com/rancher/agent/utilities/constants"
)

type DiskCollector struct {
	Unit                float64
	DataGetter          DiskInfoGetter
	InfoData            model.InfoData
	DockerStorageDriver string
	MachineInfo         *info.MachineInfo
}

type DiskInfoGetter interface {
	GetDockerStorageInfo(infoData model.InfoData) map[string]interface{}
}

type DiskDataGetter struct{}

func (d DiskCollector) GetData() (map[string]interface{}, error) {
	infoData := d.InfoData
	data := map[string]interface{}{
		"fileSystems":               map[string]interface{}{},
		"mountPoints":               map[string]interface{}{},
		"dockerStorageDriverStatus": map[string]interface{}{},
		"dockerStorageDriver":       infoData.Info.Driver,
	}

	mfs, err := d.getMachineFilesystems(infoData)
	if err != nil {
		return data, errors.Wrap(err, constants.DiskGetDataError+"failed get filesystem info")
	}
	for key, value := range mfs {
		data["fileSystems"].(map[string]interface{})[key] = value
	}
	mtp, err := d.getMountPoints()
	if err != nil {
		return data, errors.Wrap(err, constants.DiskGetDataError+"failed get mountPoint info")
	}
	for key, value := range mtp {
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
