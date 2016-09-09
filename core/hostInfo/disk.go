package hostInfo

import (
	"github.com/pkg/errors"
	"github.com/rancher/agent/model"
	"github.com/rancher/agent/utilities/constants"
)

type DiskCollector struct {
	Unit                float64
	Cadvisor            CadvisorAPIClient
	DataGetter          DiskInfoGetter
	InfoData            model.InfoData
	DockerStorageDriver string
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

	mfs, err := d.getMachineFilesystemsCadvisor(infoData)
	if err != nil {
		return data, errors.Wrap(err, constants.DiskGetDataError+"failed get filesystem info from cadvisor")
	}
	for key, value := range mfs {
		data["fileSystems"].(map[string]interface{})[key] = value
	}
	for key, value := range d.getMountPointsCadvisor() {
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
