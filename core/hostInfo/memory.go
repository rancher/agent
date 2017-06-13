package hostInfo

import (
	"github.com/pkg/errors"
	"github.com/rancher/agent/utilities/config"
	"github.com/rancher/agent/utilities/constants"
	"github.com/shirou/gopsutil/mem"
	"time"
)

type MemoryCollector struct {
	Unit      float64
	cacheData map[string]interface{}
	lastRead  time.Time
}

func (m *MemoryCollector) getMemInfoData() (map[string]interface{}, error) {
	data := map[string]interface{}{}
	memData, err := mem.VirtualMemory()
	if err != nil {
		return map[string]interface{}{}, err
	}
	data["memTotal"] = m.convertUnits(memData.Total)
	return data, nil
}

func (m *MemoryCollector) GetData() (map[string]interface{}, error) {
	if m.cacheData != nil && time.Now().Before(m.lastRead.Add(time.Minute*time.Duration(config.RefreshInterval()))) {
		return m.cacheData, nil
	}
	data, err := m.getMemInfoData()
	if err != nil {
		return map[string]interface{}{}, errors.Wrap(err, constants.MemoryGetDataError+"failed to get data")
	}
	m.cacheData = data
	m.lastRead = time.Now()
	return data, nil
}

func (m *MemoryCollector) KeyName() string {
	return "memoryInfo"
}

func (m *MemoryCollector) GetLabels(prefix string) (map[string]string, error) {
	return map[string]string{}, nil
}

func (m *MemoryCollector) convertUnits(metric uint64) uint64 {
	return metric / uint64(m.Unit*m.Unit)
}
