package hostInfo

import (
	"github.com/pkg/errors"
	"github.com/rancher/agent/utils/constants"
	"github.com/shirou/gopsutil/mem"
)

type MemoryCollector struct {
	Unit float64
}

func (m MemoryCollector) getMemInfoData() (map[string]interface{}, error) {
	data := map[string]interface{}{}
	memData, err := mem.VirtualMemory()
	if err != nil {
		return map[string]interface{}{}, err
	}
	swapData, err := mem.SwapMemory()
	if err != nil {
		return map[string]interface{}{}, err
	}
	data["memTotal"] = m.convertUnits(memData.Total)
	data["memFree"] = m.convertUnits(memData.Free)
	data["memAvailable"] = m.convertUnits(memData.Available)
	data["buffers"] = m.convertUnits(memData.Buffers)
	data["cached"] = m.convertUnits(memData.Cached)
	data["swapCached"] = m.convertUnits(swapData.Used)
	data["active"] = m.convertUnits(memData.Active)
	data["inactive"] = m.convertUnits(memData.Inactive)
	data["swaptotal"] = m.convertUnits(swapData.Total)
	data["swapfree"] = m.convertUnits(swapData.Free)
	return data, nil
}

func (m MemoryCollector) GetData() (map[string]interface{}, error) {
	data, err := m.getMemInfoData()
	if err != nil {
		return map[string]interface{}{}, errors.Wrap(err, constants.MemoryGetDataError+"failed to get data")
	}
	return data, nil
}

func (m MemoryCollector) KeyName() string {
	return "memoryInfo"
}

func (m MemoryCollector) GetLabels(prefix string) (map[string]string, error) {
	return map[string]string{}, nil
}

func (m MemoryCollector) convertUnits(metric uint64) uint64 {
	return metric / uint64(m.Unit*m.Unit)
}
