// +build darwin, windows

package handlers

import (
	"github.com/docker/engine-api/types"
	"github.com/rancher/agent/core/hostInfo"
	"github.com/rancher/agent/model"
	"runtime"
)

func getCollectors(info types.Info, version types.Version) []hostInfo.Collector {
	Collectors := []hostInfo.Collector{
		hostInfo.CPUCollector{
			DataGetter: hostInfo.CPUDataGetter{},
			GOOS:       runtime.GOOS,
		},
		hostInfo.DiskCollector{
			Unit:       1048576,
			DataGetter: hostInfo.DiskDataGetter{},
			InfoData: model.InfoData{
				Info:    info,
				Version: version,
			},
		},
		hostInfo.IopsCollector{
			GOOS: runtime.GOOS,
		},
		hostInfo.MemoryCollector{
			Unit:       1024.00,
			DataGetter: hostInfo.MemoryDataGetter{},
			GOOS:       runtime.GOOS,
		},
		hostInfo.OSCollector{
			DataGetter: hostInfo.OSDataGetter{},
			GOOS:       runtime.GOOS,
			InfoData: model.InfoData{
				Info:    info,
				Version: version,
			},
		},
	}
	return Collectors
}
