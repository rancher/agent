package hostInfo

import (
	"github.com/pkg/errors"
	"github.com/rancher/agent/model"
	"github.com/rancher/agent/utilities/constants"
	"github.com/rancher/agent/utilities/utils"
	"math"
	"strings"
)

func (d DiskCollector) convertUnits(number float64) float64 {
	// return in MB
	return math.Floor(number/d.Unit*1000) / 1000
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
			poolName = poolName[len(poolName)-5 : len(poolName)]
		}
		if strings.Contains(device, utils.InterfaceToString(poolName)) {
			include = false
		}
	}
	return include
}

func (d DiskCollector) getMountPointsCadvisor() map[string]interface{} {
	data := map[string]interface{}{}
	stat := d.Cadvisor.GetLatestStat()

	if _, ok := stat["filesystem"]; ok {
		for _, fs := range utils.InterfaceToArray(stat["filesystem"]) {
			fs := utils.InterfaceToMap(fs)
			device := utils.InterfaceToString(fs["device"])
			percentUsed := utils.InterfaceToFloat(fs["usage"]) / utils.InterfaceToFloat(fs["capacity"]) * 100
			data[device] = map[string]interface{}{
				"free":       d.convertUnits(utils.InterfaceToFloat(fs["capacity"]) - utils.InterfaceToFloat(fs["usage"])),
				"total":      d.convertUnits(utils.InterfaceToFloat(fs["usage"])),
				"used":       d.convertUnits(utils.InterfaceToFloat(fs["usage"])),
				"percentage": math.Floor(percentUsed*100) / 100,
			}
		}
	}
	return data
}

func (d DiskCollector) getMachineFilesystemsCadvisor(infoData model.InfoData) (map[string]interface{}, error) {
	data := map[string]interface{}{}
	machineInfo, err := d.Cadvisor.DataGetter.GetMachineStats()
	if err != nil {
		return data, errors.Wrap(err, constants.GetMachineFilesystemsCadvisorError)
	}
	if _, ok := machineInfo["filesystems"]; ok {
		for _, fs := range utils.InterfaceToArray(machineInfo["filesystems"]) {
			filesystem := utils.InterfaceToMap(fs)
			if d.includeInFilesystem(infoData, utils.InterfaceToString(filesystem["device"])) {
				data[utils.InterfaceToString(filesystem["device"])] = map[string]interface{}{
					"capacity": d.convertUnits(utils.InterfaceToFloat(filesystem["capacity"])),
				}
			}
		}
	}
	return data, nil
}
