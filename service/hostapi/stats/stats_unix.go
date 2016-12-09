//+build !windows

package stats

import (
	"encoding/json"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

func convertDockerStats(stats DockerStats, pid int) containerStats {
	containerStats := containerStats{}
	containerStats.Timestamp = stats.Read
	containerStats.CPU.Usage.Total = uint64(stats.CPUStats.CPUUsage.TotalUsage)
	containerStats.CPU.Usage.PerCPU = []uint64{}
	for _, value := range stats.CPUStats.CPUUsage.PercpuUsage {
		containerStats.CPU.Usage.PerCPU = append(containerStats.CPU.Usage.PerCPU, uint64(value))
	}
	containerStats.CPU.Usage.System = uint64(stats.CPUStats.CPUUsage.UsageInKernelmode)
	containerStats.CPU.Usage.User = uint64(stats.CPUStats.CPUUsage.UsageInKernelmode)
	containerStats.Memory.Usage = uint64(stats.MemoryStats.Usage)
	containerStats.Network.Interfaces = []InterfaceStats{}
	for name, netStats := range getLinkStats(pid) {
		data := InterfaceStats{}
		data.Name = name
		data.RxBytes = uint64(netStats.RxBytes)
		data.RxDropped = uint64(netStats.RxDropped)
		data.RxErrors = uint64(netStats.RxErrors)
		data.RxPackets = uint64(netStats.RxPackets)
		data.TxBytes = uint64(netStats.TxBytes)
		data.TxDropped = uint64(netStats.TxDropped)
		data.TxPackets = uint64(netStats.TxPackets)
		data.TxErrors = uint64(netStats.TxErrors)
		containerStats.Network.Interfaces = append(containerStats.Network.Interfaces, data)
	}
	containerStats.DiskIo.IoServiceBytes = []PerDiskStats{}
	for _, diskStats := range stats.BlkioStats.IoServiceBytesRecursive {
		data := PerDiskStats{}
		data.Stats = map[string]uint64{}
		data.Stats[diskStats.Op] = uint64(diskStats.Value)
		containerStats.DiskIo.IoServiceBytes = append(containerStats.DiskIo.IoServiceBytes, data)
	}
	return containerStats
}

func FromString(rawstring string) (DockerStats, error) {
	obj := DockerStats{}
	err := json.Unmarshal([]byte(rawstring), &obj)
	if err != nil {
		return obj, err
	}
	return obj, nil
}

func getLinkStats(pid int) map[string]*netlink.LinkStatistics {
	ret := map[string]*netlink.LinkStatistics{}
	nsHandler, err := netns.GetFromPid(pid)
	if err != nil {
		return nil
	}
	defer nsHandler.Close()
	handler, err := netlink.NewHandleAt(nsHandler)
	if err != nil {
		return nil
	}
	defer handler.Delete()
	links, err := handler.LinkList()
	if err != nil {
		return nil
	}
	for _, link := range links {
		attr := link.Attrs()
		if attr.Name != "lo" {
			ret[attr.Name] = attr.Statistics
		}
	}
	return ret
}
