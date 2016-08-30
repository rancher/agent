package hostInfo

import (
	"github.com/Sirupsen/logrus"
	"github.com/rancher/agent/utilities/utils"
	"math"
	"runtime"
	"strings"
)

type DiskCollector struct {
	unit                float64
	cadvisor            CadvisorAPIClient
	dockerStorageDriver string
	dataGetter          DiskInfoGetter
}

type DiskInfoGetter interface {
	GetDockerStorageInfo() map[string]interface{}
}

type DiskDataGetter struct{}

func (d DiskCollector) convertUnits(number float64) float64 {
	// return in MB
	return math.Floor(number/d.unit*1000) / 1000
}

func (d DiskDataGetter) GetDockerStorageInfo() map[string]interface{} {
	data := map[string]interface{}{}

	info := utils.GetInfo()
	for _, item := range info.DriverStatus {
		data[item[0]] = item[1]
	}
	return data
}

func (d DiskCollector) includeInFilesystem(device string) bool {
	include := true
	if d.dockerStorageDriver == "devicemapper" {
		pool := d.dataGetter.GetDockerStorageInfo()
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
	stat := d.cadvisor.GetLatestStat()

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

func (d DiskCollector) getMachineFilesystemsCadvisor() map[string]interface{} {
	data := map[string]interface{}{}
	machineInfo, err := d.cadvisor.dataGetter.GetMachineStats()
	if err == nil {
		if _, ok := machineInfo["filesystems"]; ok {
			for _, fs := range utils.InterfaceToArray(machineInfo["filesystems"]) {
				filesystem := utils.InterfaceToMap(fs)
				if d.includeInFilesystem(utils.InterfaceToString(filesystem["device"])) {
					data[utils.InterfaceToString(filesystem["device"])] = map[string]interface{}{
						"capacity": d.convertUnits(utils.InterfaceToFloat(filesystem["capacity"])),
					}
				}
			}
		}
	}
	return data
}

func (d DiskCollector) GetData() map[string]interface{} {
	data := map[string]interface{}{
		"fileSystems":               map[string]interface{}{},
		"mountPoints":               map[string]interface{}{},
		"dockerStorageDriverStatus": map[string]interface{}{},
		"dockerStorageDriver":       d.dockerStorageDriver,
	}
	if runtime.GOOS == "linux" {
		for key, value := range d.getMachineFilesystemsCadvisor() {
			data["fileSystems"].(map[string]interface{})[key] = value
		}
		for key, value := range d.getMountPointsCadvisor() {
			data["mountPoints"].(map[string]interface{})[key] = value
		}
	}
	return data
}

func (d DiskCollector) KeyName() string {
	return "diskInfo"
}

func (d DiskCollector) GetLabels(prefix string) map[string]string {
	return map[string]string{}
}
