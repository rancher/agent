package model

import (
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/container"
	"github.com/docker/go-units"
)

type Instance struct {
	AccountID                   int    `json:"accountId"`
	AllocationState             string `json:"allocationState"`
	Created                     int64  `json:"created"`
	Data                        InstanceFieldsData
	Ports                       []Port
	Description                 string `json:"description"`
	Hostname                    string `json:"hostname"`
	ID                          int    `json:"id"`
	Image                       Image
	ImageID                     int    `json:"imageId"`
	Kind                        string `json:"kind"`
	Name                        string `json:"name"`
	Nics                        []Nic
	Offering                    interface{} `json:"offering"`
	OfferingID                  interface{} `json:"offeringId"`
	OnCrash                     string      `json:"onCrash"`
	PostComputeState            string      `json:"postComputeState"`
	RemoveTime                  interface{} `json:"removeTime"`
	Removed                     interface{} `json:"removed"`
	RequestedOfferingID         interface{} `json:"requestedOfferingId"`
	RequestedState              interface{} `json:"requestedState"`
	State                       string      `json:"state"`
	Type                        string      `json:"type"`
	UUID                        string      `json:"uuid"`
	Volumes                     []Volume
	ZoneID                      int `json:"zoneId"`
	ExternalID                  string
	AgentID                     int
	InstanceLinks               []Link
	NetworkContainer            *Instance
	NativeContainer             bool
	SystemContainer             string
	DataVolumesFromContainers   []*Instance
	CommandArgs                 []string
	Labels                      map[string]interface{}
	ProcessData                 ProcessData
	VolumesFromDataVolumeMounts []Volume
	Token                       string
}

type InstanceFieldsData struct {
	Fields        InstanceFields
	IPSec         IPSec
	DockerInspect ContainerJSON `json:"dockerInspect,omitempty" yaml:"dockerInspect,omitempty"`
	Process       ProcessData
}

type ContainerJSON struct {
	ID              string `json:"Id"`
	Created         string
	Path            string
	Args            []string
	State           types.ContainerState
	Image           string
	ResolvConfPath  string
	HostnamePath    string
	HostsPath       string
	LogPath         string
	Node            types.ContainerNode `json:",omitempty"`
	Name            string
	RestartCount    int
	Driver          string
	MountLabel      string
	ProcessLabel    string
	AppArmorProfile string
	ExecIDs         []string
	HostConfig      *container.HostConfig
	GraphDriver     types.GraphDriverData
	SizeRw          int64 `json:",omitempty"`
	SizeRootFs      int64 `json:",omitempty"`
	Mounts          []types.MountPoint
	Config          container.Config
	NetworkSettings types.NetworkSettings
}

type InstanceFields struct {
	PublishAllPorts    bool
	DataVolumes        []string
	Privileged         bool
	ReadOnly           bool
	BlkioDeviceOptions map[string]DeviceOptions
	CommandArgs        []string
	ExtraHosts         []string
	PidMode            container.PidMode
	LogConfig          LogConfig
	SecurityOpt        []string
	Devices            []string
	DNS                []string
	DNSSearch          []string
	CapAdd             []string
	CapDrop            []string
	RestartPolicy      container.RestartPolicy
	CPUShares          int64
	VolumeDriver       string
	CPUSet             string
	BlkioWeight        uint16
	CgroupParent       string
	CPUPeriod          int64
	CPUQuota           int64
	CPUsetMems         string
	DNSOpt             []string
	GroupAdd           []string
	Isolation          container.Isolation
	KernelMemory       int64
	Memory             int64
	MemoryReservation  int64
	MemorySwap         int64
	MemorySwappiness   *int64
	OomKillDisable     *bool
	ShmSize            int64
	Tmpfs              map[string]string
	Ulimits            []*units.Ulimit
	Uts                container.UTSMode
	IpcMode            container.IpcMode
	ConsoleSize        [2]int
	CPUCount           int64
	CPUPercent         int64
	IOMaximumIOps      uint64
	IOMaximumBandwidth uint64
	Command            interface{} // this one is so weird
	Environment        map[string]string
	WorkingDir         string
	EntryPoint         []string
	Tty                bool
	StdinOpen          bool
	DomainName         string
	Labels             map[string]string
	StopSignal         string
}

type LogConfig struct {
	Driver string
	Config map[string]string
}

type DeviceOptions struct {
	Weight    uint16
	ReadIops  uint64
	WriteIops uint64
	ReadBps   uint64
	WriteBps  uint64
}

type IPSec struct {
	Setting map[string]struct {
		Nat    float64
		Isakmp float64
	}
}
