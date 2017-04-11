package hostInfo

import (
	"github.com/pkg/errors"
	"github.com/rancher/agent/model"
	"github.com/rancher/agent/utilities/config"
	"github.com/rancher/agent/utilities/constants"
	"time"
)

type DiskCollector struct {
	Unit                uint64
	InfoData            model.InfoData
	DockerStorageDriver string
	cacheData           map[string]interface{}
	lastRead            time.Time
}

func (d *DiskCollector) GetData() (map[string]interface{}, error) {
	if d.cacheData != nil && time.Now().Before(d.lastRead.Add(time.Minute*time.Duration(config.RefreshInterval()))) {
		return d.cacheData, nil
	}

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
	mp, err := d.getMountPoints()
	if err != nil {
		return data, errors.Wrap(err, constants.DiskGetDataError+"failed get mountpoint info")
	}
	for key, value := range mp {
		data["mountPoints"].(map[string]interface{})[key] = value
	}
	d.cacheData = data
	d.lastRead = time.Now()
	return data, nil
}

func (d *DiskCollector) KeyName() string {
	return "diskInfo"
}

func (d *DiskCollector) GetLabels(prefix string) (map[string]string, error) {
	return map[string]string{}, nil
}
