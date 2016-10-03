package hostInfo

import (
	"fmt"
	"github.com/shirou/gopsutil/cpu"
	"testing"
)

func TestCpuTime(t *testing.T) {
	cpuPrevStat, err := cpu.Times(false)
	if err != nil {
		t.Fatal(err)
	}
	for _, stat := range cpuPrevStat {
		fmt.Printf("%+v", stat)
	}
}

func TestCpuInfo(t *testing.T) {
	cpuInfo, err := cpu.Info()
	if err != nil {
		t.Fatal(err)
	}
	for _, info := range cpuInfo {
		fmt.Printf("%+v", info)
	}
}
