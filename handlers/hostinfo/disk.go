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
		if strings.HasSuffix(poolName.(string), "-pool") {
			poolName := poolName.(string)
			poolName = poolName[len(poolName)-5 : len(poolName)]
		}
		if strings.Contains(device, poolName.(string)) {
			include = false
		}
	}
	return include
}

func (d DiskCollector) getMountpointsCadvisor() map[string]interface{} {
	data := map[string]interface{}{}
	stat := d.cadvisor.GetLatestStat()

	if _, ok := stat["filesystem"]; ok {
		for _, fs := range stat["filesystem"].([]map[string]interface{}) {
			device := fs["device"].(string)
			percentUsed := fs["usage"].(float64) / fs["capacity"].(float64) * 100
			data[device] = map[string]interface{}{
				"free":       d.convertUnits(fs["capacity"].(float64) - fs["usage"].(float64)),
				"total":      d.convertUnits(fs["uasge"].(float64)),
				"used":       d.convertUnits(fs["usage"].(float64)),
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
		for _, filesystem := range machineInfo["filesystems"].([]map[string]interface{}) {
			if d.includeInFilesystem(filesystem["device"].(string)) {
				data[filesystem["device"].(string)] = map[string]interface{}{
					"capacity": d.convertUnits(filesystem["capacity"].(float64)),
				}
			}
		}
	}
	return data
}

func (d DiskCollector) GetData() map[string]interface{} {
	data := map[string]interface{}{
		"fileSystem":                map[string]interface{}{},
		"mountPoints":               map[string]interface{}{},
		"dockerStorageDriverStatus": map[string]interface{}{},
		"dockerStorageDriver":       d.dockerStorageDriver,
	}

	if runtime.GOOS == "linux" {
		for key, value := range d.getMachineFilesystemsCadvisor() {
			data["fileSystems"].(map[string]interface{})[key] = value
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
