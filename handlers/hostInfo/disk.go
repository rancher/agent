package hostInfo

import (
	"github.com/docker/engine-api/client"
	"golang.org/x/net/context"
	"math"
	"runtime"
	"strings"
)

type DiskCollector struct {
	unit                float64
	cadvisor            CadvisorAPIClient
	client              *client.Client
	dockerStorageDriver string
}

func (d DiskCollector) convertUnits(number float64) float64 {
	// return in MB
	return math.Floor(number/d.unit*1000) / 1000
}

func (d DiskCollector) getDockerStorageInfo() map[string]interface{} {
	data := map[string]interface{}{}

	if d.client != nil {
		info, _ := d.client.Info(context.Background())
		for _, item := range info.DriverStatus {
			data[item[0]] = item[1]
		}
	}
	return data
}

func (d DiskCollector) includeInFilesystem(device string) bool {
	include := true
	if d.dockerStorageDriver == "devicemapper" {
		pool := d.getDockerStorageInfo()
		poolName, ok := pool["Pool Name"]
		if !ok {
			poolName = "/dev/mapper/docker-"
		}
		if strings.HasSuffix(InterfaceToString(poolName), "-pool") {
			poolName := InterfaceToString(poolName)
			poolName = poolName[len(poolName)-5 : len(poolName)]
		}
		if strings.Contains(device, InterfaceToString(poolName)) {
			include = false
		}
	}
	return include
}

func (d DiskCollector) getMountpointsCadvisor() map[string]interface{} {
	data := map[string]interface{}{}
	stat := d.cadvisor.GetLatestStat()

	if _, ok := stat["filesystem"]; ok {
		for _, fs := range InterfaceToArray(stat["filesystem"]) {
			fs := InterfaceToMap(fs)
			device := InterfaceToString(fs["device"])
			percentUsed := InterfaceToFloat(fs["usage"]) / InterfaceToFloat(fs["capacity"]) * 100
			data[device] = map[string]interface{}{
				"free":       d.convertUnits(InterfaceToFloat(fs["capacity"]) - InterfaceToFloat(fs["usage"])),
				"total":      d.convertUnits(InterfaceToFloat(fs["usage"])),
				"used":       d.convertUnits(InterfaceToFloat(fs["usage"])),
				"percentage": math.Floor(percentUsed*100) / 100,
			}
		}
	}
	return data
}

func (d DiskCollector) getMachineFilesystemsCadvisor() map[string]interface{} {
	data := map[string]interface{}{}
	machineInfo := d.cadvisor.GetMachineStats()

	if _, ok := machineInfo["filesystems"]; ok {
		for _, fs := range InterfaceToArray(machineInfo["filesystems"]) {
			filesystem := InterfaceToMap(fs)
			if d.includeInFilesystem(InterfaceToString(filesystem["device"])) {
				data[InterfaceToString(filesystem["device"])] = map[string]interface{}{
					"capacity": d.convertUnits(InterfaceToFloat(filesystem["capacity"])),
				}
			}
		}
	}
	return data
}

func (d DiskCollector) GetData() map[string]interface{} {
	data := map[string]interface{}{
		"filesystems":                map[string]interface{}{},
		"mountPoints":               map[string]interface{}{},
		"dockerStorageDriverStatus": map[string]interface{}{},
		"dockerStorageDriver":       d.dockerStorageDriver,
	}

	if runtime.GOOS == "linux" {
		for key, value := range d.getMachineFilesystemsCadvisor() {
			data["filesystems"].(map[string]interface{})[key] = value
		}
		for key, value := range d.getMountpointsCadvisor() {
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
