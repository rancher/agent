package model

import (
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/blkiodev"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/go-connections/nat"
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
	ZoneID                      int    `json:"zoneId"`
	ExternalID                  string `json:"externalId"`
	AgentID                     int
	InstanceLinks               []Link
	NetworkContainer            *Instance
	NativeContainer             bool
	DataVolumesFromContainers   []*Instance
	CommandArgs                 []string
	Labels                      map[string]interface{}
	ProcessData                 ProcessData
	VolumesFromDataVolumeMounts []Volume
	Token                       string
	MilliCPUReservation         int64
	MemoryReservation           int64
	System                      bool
	ImageCredential             RegistryCredential
}

type InstanceFieldsData struct {
	Fields InstanceFields
	IPSec  map[string]struct {
		Nat    float64
		Isakmp float64
	} `json:"ipsec"`
	DockerInspect ContainerJSON `json:"dockerInspect,omitempty" yaml:"dockerInspect,omitempty"`
	Process       ProcessData
}

type ContainerJSON struct {
	ID              string `json:"Id"`
	Created         string
	Path            string
	Args            []string
	State           interface{}
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
	HostConfig      HostConfig
	GraphDriver     types.GraphDriverData
	SizeRw          int64 `json:",omitempty"`
	SizeRootFs      int64 `json:",omitempty"`
	Mounts          []types.MountPoint
	Config          container.Config
	NetworkSettings types.NetworkSettings
}

type HostConfig struct {
	// Applicable to all platforms
	Binds           []string                // List of volume bindings for this container
	ContainerIDFile string                  // File (path) where the containerId is written
	LogConfig       LogConfig               // Configuration of the logs for this container
	NetworkMode     container.NetworkMode   // Network mode to use for the container
	PortBindings    nat.PortMap             // Port mapping between the exposed port (container) and the host
	RestartPolicy   container.RestartPolicy // Restart policy to be used for the container
	AutoRemove      bool                    // Automatically remove container when it exits
	VolumeDriver    string                  // Name of the volume driver used to mount volumes
	VolumesFrom     []string                // List of volumes to take from other container

	// Applicable to UNIX platforms
	CapAdd          strslice.StrSlice    // List of kernel capabilities to add to the container
	CapDrop         strslice.StrSlice    // List of kernel capabilities to remove from the container
	DNS             []string             `json:"Dns"`        // List of DNS server to lookup
	DNSOptions      []string             `json:"DnsOptions"` // List of DNSOption to look for
	DNSSearch       []string             `json:"DnsSearch"`  // List of DNSSearch to look for
	ExtraHosts      []string             // List of extra hosts
	GroupAdd        []string             // List of additional groups that the container process will run as
	IpcMode         container.IpcMode    // IPC namespace to use for the container
	Cgroup          container.CgroupSpec // Cgroup to use for the container
	Links           []string             // List of links (in the name:alias form)
	OomScoreAdj     int                  // Container preference for OOM-killing
	PidMode         container.PidMode    // PID namespace to use for the container
	Privileged      bool                 // Is the container in privileged mode
	PublishAllPorts bool                 // Should docker publish all exposed port for the container
	ReadonlyRootfs  bool                 // Is the container root filesystem in read-only
	SecurityOpt     []string             // List of string values to customize labels for MLS systems, such as SELinux.
	StorageOpt      map[string]string    `json:",omitempty"` // Storage driver options per container.
	Tmpfs           map[string]string    `json:",omitempty"` // List of tmpfs (mounts) used for the container
	UTSMode         container.UTSMode    // UTS namespace to use for the container
	UsernsMode      container.UsernsMode // The user namespace to use for the container
	ShmSize         int64                // Total shm memory usage
	Sysctls         map[string]string    `json:",omitempty"` // List of Namespaced sysctls used for the container
	Init            *bool                `json:",omitempty"` // Should init be run in the container
	Runtime         string               `json:",omitempty"` // Runtime to use with this container

	// Applicable to Windows
	//ConsoleSize [2]int    // Initial console size
	Isolation container.Isolation // Isolation technology of the container (eg default, hyperv)

	// Contains container's resources (cgroups, ulimits)
	Resources

	// Mounts specs used by the container
	Mounts []mount.Mount `json:",omitempty"`
}

type Resources struct {
	// Applicable to all platforms
	CPUShares int64 `json:"CpuShares"` // CPU shares (relative weight vs. other containers)
	Memory    int64 // Memory limit (in bytes)

	// Applicable to UNIX platforms
	CgroupParent         string // Parent cgroup.
	BlkioWeight          uint16 // Block IO weight (relative weight vs. other containers)
	BlkioWeightDevice    []*blkiodev.WeightDevice
	BlkioDeviceReadBps   []*blkiodev.ThrottleDevice
	BlkioDeviceWriteBps  []*blkiodev.ThrottleDevice
	BlkioDeviceReadIOps  []*blkiodev.ThrottleDevice
	BlkioDeviceWriteIOps []*blkiodev.ThrottleDevice
	CPUPeriod            int64                     `json:"CpuPeriod"` // CPU CFS (Completely Fair Scheduler) period
	CPUQuota             int64                     `json:"CpuQuota"`  // CPU CFS (Completely Fair Scheduler) quota
	CpusetCpus           string                    // CpusetCpus 0-2, 0,1
	CpusetMems           string                    // CpusetMems 0-2, 0,1
	Devices              []container.DeviceMapping // List of devices to map inside the container
	DiskQuota            int64                     // Disk limit (in bytes)
	KernelMemory         int64                     // Kernel memory limit (in bytes)
	MemoryReservation    int64                     // Memory soft limit (in bytes)
	MemorySwap           int64                     // Total memory usage (memory + swap); set `-1` to enable unlimited swap
	MemorySwappiness     *int64                    // Tuning container memory swappiness behaviour
	OomKillDisable       *bool                     // Whether to disable OOM Killer or not
	PidsLimit            int64                     // Setting pids limit for a container
	Ulimits              []*units.Ulimit           // List of ulimits to be set in the container

	// Applicable to Windows
	CPUCount           int64  `json:"CpuCount"`   // CPU count
	CPUPercent         int64  `json:"CpuPercent"` // CPU percent
	IOMaximumIOps      uint64 // Maximum IOps for the container system drive
	IOMaximumBandwidth uint64 // Maximum IO in bytes per second for the container system drive
}

type InstanceFields struct {
	ImageUUID          string `json:"imageUuid"`
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
	DNS                []string `json:"dns"`
	DNSSearch          []string `json:"dnsSearch"`
	CapAdd             []string
	CapDrop            []string
	RestartPolicy      container.RestartPolicy
	CPUShares          int64 `json:"cpuShares"`
	VolumeDriver       string
	CPUSet             string
	BlkioWeight        uint16
	CgroupParent       string
	CPUPeriod          int64  `json:"cpuPeriod"`
	CPUQuota           int64  `json:"cpuQuota"`
	CPUsetMems         string `json:"cpuSetMems"`
	DNSOpt             []string
	GroupAdd           []string
	Isolation          container.Isolation
	KernelMemory       int64
	Memory             int64
	MemorySwap         int64
	MemorySwappiness   *int64
	OomKillDisable     *bool
	OomScoreAdj        int
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
	User               string
	Sysctls            map[string]string
	Init               *bool
	HealthCmd          []string
	HealthTimeout      int
	HealthInterval     int
	HealthRetries      int
	StorageOpt         map[string]string
	PidsLimit          int64
	Cgroup             container.CgroupSpec
	DiskQuota          int64
	UsernsMode         container.UsernsMode
	Build              BuildOptions
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
