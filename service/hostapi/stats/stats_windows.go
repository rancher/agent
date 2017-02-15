package stats

import (
	"encoding/json"
	"time"
)

type DockerStatsWindows struct {
	Read      time.Time `json:"read"`
	Preread   time.Time `json:"preread"`
	PidsStats struct {
	} `json:"pids_stats"`
	BlkioStats struct {
		IoServiceBytesRecursive interface{} `json:"io_service_bytes_recursive"`
		IoServicedRecursive     interface{} `json:"io_serviced_recursive"`
		IoQueueRecursive        interface{} `json:"io_queue_recursive"`
		IoServiceTimeRecursive  interface{} `json:"io_service_time_recursive"`
		IoWaitTimeRecursive     interface{} `json:"io_wait_time_recursive"`
		IoMergedRecursive       interface{} `json:"io_merged_recursive"`
		IoTimeRecursive         interface{} `json:"io_time_recursive"`
		SectorsRecursive        interface{} `json:"sectors_recursive"`
	} `json:"blkio_stats"`
	NumProcs     int `json:"num_procs"`
	StorageStats struct {
		ReadCountNormalized  int `json:"read_count_normalized"`
		ReadSizeBytes        int `json:"read_size_bytes"`
		WriteCountNormalized int `json:"write_count_normalized"`
		WriteSizeBytes       int `json:"write_size_bytes"`
	} `json:"storage_stats"`
	CPUStats struct {
		CPUUsage struct {
			TotalUsage        int `json:"total_usage"`
			UsageInKernelmode int `json:"usage_in_kernelmode"`
			UsageInUsermode   int `json:"usage_in_usermode"`
		} `json:"cpu_usage"`
		ThrottlingData struct {
			Periods          int `json:"periods"`
			ThrottledPeriods int `json:"throttled_periods"`
			ThrottledTime    int `json:"throttled_time"`
		} `json:"throttling_data"`
	} `json:"cpu_stats"`
	PrecpuStats struct {
		CPUUsage struct {
			TotalUsage        int `json:"total_usage"`
			UsageInKernelmode int `json:"usage_in_kernelmode"`
			UsageInUsermode   int `json:"usage_in_usermode"`
		} `json:"cpu_usage"`
		ThrottlingData struct {
			Periods          int `json:"periods"`
			ThrottledPeriods int `json:"throttled_periods"`
			ThrottledTime    int `json:"throttled_time"`
		} `json:"throttling_data"`
	} `json:"precpu_stats"`
	MemoryStats struct {
		Commitbytes       int `json:"commitbytes"`
		Commitpeakbytes   int `json:"commitpeakbytes"`
		Privateworkingset int `json:"privateworkingset"`
	} `json:"memory_stats"`
	Name     string `json:"name"`
	ID       string `json:"id"`
	Networks map[string]struct {
		RxBytes   int `json:"rx_bytes"`
		RxPackets int `json:"rx_packets"`
		RxErrors  int `json:"rx_errors"`
		RxDropped int `json:"rx_dropped"`
		TxBytes   int `json:"tx_bytes"`
		TxPackets int `json:"tx_packets"`
		TxErrors  int `json:"tx_errors"`
		TxDropped int `json:"tx_dropped"`
	} `json:"networks"`
}

func convertDockerStats(stats DockerStatsWindows, pid int) containerStats {
	containerStats := containerStats{}
	containerStats.Timestamp = stats.Read
	containerStats.CPU.Usage.Total = uint64(stats.CPUStats.CPUUsage.TotalUsage)
	containerStats.CPU.Usage.System = uint64(stats.CPUStats.CPUUsage.UsageInKernelmode)
	containerStats.CPU.Usage.User = uint64(stats.CPUStats.CPUUsage.UsageInKernelmode)
	containerStats.Memory.Usage = uint64(stats.MemoryStats.Privateworkingset)
	containerStats.Network.Interfaces = []InterfaceStats{}
	for name, netStats := range stats.Networks {
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
	data := PerDiskStats{}
	data.Stats = map[string]uint64{}
	data.Stats["Read"] = uint64(stats.StorageStats.ReadSizeBytes)
	data.Stats["Write"] = uint64(stats.StorageStats.WriteSizeBytes)
	containerStats.DiskIo.IoServiceBytes = append(containerStats.DiskIo.IoServiceBytes, data)
	return containerStats
}

func convertStatsFromRaw(raw []byte) (DockerStatsWindows, error) {
	obj := DockerStatsWindows{}
	err := json.Unmarshal(raw, &obj)
	if err != nil {
		return obj, err
	}
	return obj, nil
}
