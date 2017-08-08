package hostInfo

import (
	"math"
	"strings"

	"github.com/rancher/agent/utils"
	"github.com/shirou/gopsutil/disk"
)

func (d DiskCollector) convertUnits(number uint64) float64 {
	// return in MB
	return math.Floor(float64(number/d.Unit*1000)) / 1000.0
}

func (d DiskCollector) getDockerStorageInfo() map[string]interface{} {
	data := map[string]interface{}{}

	info := DockerData.Info
	for _, item := range info.DriverStatus {
		data[item[0]] = item[1]
	}
	return data
}

func (d DiskCollector) includeInFilesystem(device string) bool {
	include := true
	if DockerData.Info.Driver == "devicemapper" {
		pool := d.getDockerStorageInfo()
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

func (d DiskCollector) getMountPoints() (map[string]interface{}, error) {
	data := map[string]interface{}{}
	partitions, err := disk.Partitions(false)
	if err != nil {
		return data, err
	}
	for _, partition := range partitions {
		usage, err := disk.Usage(partition.Mountpoint)
		if err != nil {
			return map[string]interface{}{}, err
		}
		data[partition.Device] = map[string]interface{}{
			"free":       d.convertUnits(usage.Free),
			"total":      d.convertUnits(usage.Total),
			"used":       d.convertUnits(usage.Used),
			"percentage": usage.UsedPercent,
		}
	}
	return data, nil
}

func (d DiskCollector) getMachineFilesystems() (map[string]interface{}, error) {
	data := map[string]interface{}{}
	partitions, err := disk.Partitions(false)
	if err != nil {
		return data, err
	}
	for _, partition := range partitions {
		if d.includeInFilesystem(partition.Device) {
			usage, err := disk.Usage(partition.Mountpoint)
			if err != nil {
				return map[string]interface{}{}, err
			}
			data[utils.InterfaceToString(partition.Device)] = map[string]interface{}{
				"capacity": d.convertUnits(usage.Total),
			}
		}
	}
	return data, nil
}
