// +build linux freebsd solaris openbsd darwin

package hostInfo

import (
	"github.com/pkg/errors"
	"github.com/rancher/agent/model"
	"github.com/rancher/agent/utilities/constants"
	"github.com/rancher/agent/utilities/utils"
	"math"
	"strings"
)

type FileSystem struct {
	Type      string
	Capacity  uint64
	Free      uint64
	Available uint64
}

type partition struct {
	mountPoint string
	major      uint
	minor      uint
	fsType     string
	blockSize  uint
}

func (d DiskCollector) convertUnits(number uint64) float64 {
	// return in MB
	return math.Floor(float64(number)/d.Unit*1000) / 1000
}

func (d DiskDataGetter) GetDockerStorageInfo(infoData model.InfoData) map[string]interface{} {
	data := map[string]interface{}{}

	info := infoData.Info
	for _, item := range info.DriverStatus {
		data[item[0]] = item[1]
	}
	return data
}

func (d DiskCollector) includeInFilesystem(infoData model.InfoData, device string) bool {
	include := true
	if infoData.Info.Driver == "devicemapper" {
		pool := d.DataGetter.GetDockerStorageInfo(infoData)
		poolName, ok := pool["Pool Name"]
		if !ok {
			poolName = "/dev/mapper/docker-"
		}
		if strings.HasSuffix(utils.InterfaceToString(poolName), "-pool") {
			poolName := utils.InterfaceToString(poolName)
			poolName = poolName[len(poolName)-5:]
		}
		if strings.Contains(device, utils.InterfaceToString(poolName)) {
			include = false
		}
	}
	return include
}

func (d DiskCollector) getMountPoints() (map[string]interface{}, error) {
	data := map[string]interface{}{}
	fsInfo, err := GetFileInfo()
	if err != nil {
		return map[string]interface{}{}, errors.Wrap(err, constants.GetMountPointsError+"failed to get file info")
	}
	fsList, err := fsInfo.GetGlobalFsInfo()
	if err != nil {
		return map[string]interface{}{}, errors.Wrap(err, constants.GetMountPointsError+"failed to get file info")
	}
	for _, fs := range fsList {
		device := fs.Device
		if device == "none" {
			continue
		}
		usage := fs.Capacity - fs.Free
		percentUsed := usage / fs.Capacity * 100.0
		data[device] = map[string]interface{}{
			"free":       d.convertUnits(fs.Free),
			"total":      d.convertUnits(fs.Capacity),
			"used":       d.convertUnits(usage),
			"percentage": math.Floor(float64(percentUsed*100)) / 100,
		}
	}

	return data, nil
}

func (d DiskCollector) getMachineFilesystems(infoData model.InfoData) (map[string]interface{}, error) {
	data := map[string]interface{}{}
	if d.MachineInfo == nil {
		return map[string]interface{}{}, nil
	}
	machineInfo := d.MachineInfo

	for _, fs := range machineInfo.Filesystems {
		if fs.Device == "none" {
			continue
		}
		if d.includeInFilesystem(infoData, fs.Device) {
			data[fs.Device] = map[string]interface{}{
				"capacity": d.convertUnits(fs.Capacity),
			}
		}
	}

	return data, nil
}
