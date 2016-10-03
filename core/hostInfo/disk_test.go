package hostInfo

import (
	"fmt"
	"github.com/shirou/gopsutil/disk"
	"testing"
)

func TestDiskData(t *testing.T) {
	partitions, err := disk.Partitions(true)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range partitions {
		usage, err := disk.Usage(p.Mountpoint)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Printf("%+v", usage)
	}
}
