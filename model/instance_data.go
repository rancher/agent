package model

import "time"

type InstanceData struct {
	Name                  string      `json:"name"`
	ID                    int         `json:"id"`
	State                 string      `json:"state"`
	AccountID             int         `json:"accountId"`
	AgentID               int         `json:"agentId"`
	ZoneID                int         `json:"zoneId"`
	InstanceTriggeredStop string      `json:"instanceTriggeredStop"`
	RegistryCredentialID  interface{} `json:"registryCredentialId"`
	ImageID               int         `json:"imageId"`
	HealthState           interface{} `json:"healthState"`
	CreateIndex           interface{} `json:"createIndex"`
	Domain                interface{} `json:"domain"`
	Hostname              interface{} `json:"hostname"`
	Created               int64       `json:"created"`
	ExternalID            string      `json:"externalId"`
	Kind                  string      `json:"kind"`
	UUID                  string      `json:"uuid"`
	HealthUpdated         interface{} `json:"healthUpdated"`
	OfferingID            interface{} `json:"offeringId"`
	DeploymentUnitUUID    interface{} `json:"deploymentUnitUuid"`
	FirstRunning          int64       `json:"firstRunning"`
	Description           interface{} `json:"description"`
	Data                  struct {
		Fields struct {
			ImageUUID        string        `json:"imageUuid"`
			NetworkIds       []interface{} `json:"networkIds"`
			VnetIds          []int         `json:"vnetIds"`
			DataVolumes      []string      `json:"dataVolumes"`
			Privileged       bool          `json:"privileged"`
			DataVolumeMounts struct {
			} `json:"dataVolumeMounts"`
			TransitioningProgress int      `json:"transitioningProgress"`
			HostID                int      `json:"hostId"`
			DockerHostIP          string   `json:"dockerHostIp"`
			DockerIP              string   `json:"dockerIp"`
			DockerPorts           []string `json:"dockerPorts"`
			Labels                struct {
				IoRancherContainerSystem  string `json:"io.rancher.container.system"`
				IoRancherContainerUUID    string `json:"io.rancher.container.uuid"`
				IoRancherContainerName    string `json:"io.rancher.container.name"`
				IoRancherContainerAgentID string `json:"io.rancher.container.agent_id"`
				IoRancherContainerIP      string `json:"io.rancher.container.ip"`
			} `json:"labels"`
			PrimaryIPAddress     string `json:"primaryIpAddress"`
			TransitioningMessage string `json:"transitioningMessage"`
		} `json:"fields"`
		Ipsec struct {
			Num1 struct {
				Isakmp int `json:"isakmp"`
				Nat    int `json:"nat"`
			} `json:"1"`
		} `json:"ipsec"`
		DockerContainer struct {
			ID      string   `json:"Id"`
			Names   []string `json:"Names"`
			Image   string   `json:"Image"`
			ImageID string   `json:"ImageID"`
			Command string   `json:"Command"`
			Created int      `json:"Created"`
			Ports   []struct {
				IP          string `json:"IP"`
				PrivatePort int    `json:"PrivatePort"`
				PublicPort  int    `json:"PublicPort"`
				Type        string `json:"Type"`
			} `json:"Ports"`
			Labels struct {
				IoRancherContainerAgentID string `json:"io.rancher.container.agent_id"`
				IoRancherContainerIP      string `json:"io.rancher.container.ip"`
				IoRancherContainerName    string `json:"io.rancher.container.name"`
				IoRancherContainerSystem  string `json:"io.rancher.container.system"`
				IoRancherContainerUUID    string `json:"io.rancher.container.uuid"`
			} `json:"Labels"`
			State      string `json:"State"`
			Status     string `json:"Status"`
			HostConfig struct {
				NetworkMode string `json:"NetworkMode"`
			} `json:"HostConfig"`
			NetworkSettings struct {
				Networks struct {
					Bridge struct {
						IPAMConfig          interface{} `json:"IPAMConfig"`
						Links               interface{} `json:"Links"`
						Aliases             interface{} `json:"Aliases"`
						NetworkID           string      `json:"NetworkID"`
						EndpointID          string      `json:"EndpointID"`
						Gateway             string      `json:"Gateway"`
						IPAddress           string      `json:"IPAddress"`
						IPPrefixLen         int         `json:"IPPrefixLen"`
						IPv6Gateway         string      `json:"IPv6Gateway"`
						GlobalIPv6Address   string      `json:"GlobalIPv6Address"`
						GlobalIPv6PrefixLen int         `json:"GlobalIPv6PrefixLen"`
						MacAddress          string      `json:"MacAddress"`
					} `json:"bridge"`
				} `json:"Networks"`
			} `json:"NetworkSettings"`
			Mounts interface{} `json:"Mounts"`
		} `json:"dockerContainer"`
		DockerInspect struct {
			ID      string    `json:"Id"`
			Created time.Time `json:"Created"`
			Path    string    `json:"Path"`
			Args    []string  `json:"Args"`
			State   struct {
				Status     string    `json:"Status"`
				Running    bool      `json:"Running"`
				Paused     bool      `json:"Paused"`
				Restarting bool      `json:"Restarting"`
				OOMKilled  bool      `json:"OOMKilled"`
				Dead       bool      `json:"Dead"`
				Pid        int       `json:"Pid"`
				ExitCode   int       `json:"ExitCode"`
				Error      string    `json:"Error"`
				StartedAt  time.Time `json:"StartedAt"`
				FinishedAt time.Time `json:"FinishedAt"`
			} `json:"State"`
			Image           string      `json:"Image"`
			ResolvConfPath  string      `json:"ResolvConfPath"`
			HostnamePath    string      `json:"HostnamePath"`
			HostsPath       string      `json:"HostsPath"`
			LogPath         string      `json:"LogPath"`
			Name            string      `json:"Name"`
			RestartCount    int         `json:"RestartCount"`
			Driver          string      `json:"Driver"`
			MountLabel      string      `json:"MountLabel"`
			ProcessLabel    string      `json:"ProcessLabel"`
			AppArmorProfile string      `json:"AppArmorProfile"`
			ExecIDs         interface{} `json:"ExecIDs"`
			HostConfig      struct {
				Binds           []string `json:"Binds"`
				ContainerIDFile string   `json:"ContainerIDFile"`
				LogConfig       struct {
					Type   string `json:"Type"`
					Config struct {
					} `json:"Config"`
				} `json:"LogConfig"`
				NetworkMode  string `json:"NetworkMode"`
				PortBindings struct {
					Four500UDP []struct {
						HostIP   string `json:"HostIp"`
						HostPort string `json:"HostPort"`
					} `json:"4500/udp"`
					Five00UDP []struct {
						HostIP   string `json:"HostIp"`
						HostPort string `json:"HostPort"`
					} `json:"500/udp"`
				} `json:"PortBindings"`
				RestartPolicy struct {
					Name              string `json:"Name"`
					MaximumRetryCount int    `json:"MaximumRetryCount"`
				} `json:"RestartPolicy"`
				AutoRemove           bool        `json:"AutoRemove"`
				VolumeDriver         string      `json:"VolumeDriver"`
				VolumesFrom          interface{} `json:"VolumesFrom"`
				CapAdd               interface{} `json:"CapAdd"`
				CapDrop              interface{} `json:"CapDrop"`
				DNS                  interface{} `json:"Dns"`
				DNSOptions           interface{} `json:"DnsOptions"`
				DNSSearch            interface{} `json:"DnsSearch"`
				ExtraHosts           interface{} `json:"ExtraHosts"`
				GroupAdd             interface{} `json:"GroupAdd"`
				IpcMode              string      `json:"IpcMode"`
				Cgroup               string      `json:"Cgroup"`
				Links                interface{} `json:"Links"`
				OomScoreAdj          int         `json:"OomScoreAdj"`
				PidMode              string      `json:"PidMode"`
				Privileged           bool        `json:"Privileged"`
				PublishAllPorts      bool        `json:"PublishAllPorts"`
				ReadonlyRootfs       bool        `json:"ReadonlyRootfs"`
				SecurityOpt          interface{} `json:"SecurityOpt"`
				UTSMode              string      `json:"UTSMode"`
				UsernsMode           string      `json:"UsernsMode"`
				ShmSize              int         `json:"ShmSize"`
				ConsoleSize          []int       `json:"ConsoleSize"`
				Isolation            string      `json:"Isolation"`
				CPUShares            int         `json:"CpuShares"`
				Memory               int         `json:"Memory"`
				CgroupParent         string      `json:"CgroupParent"`
				BlkioWeight          int         `json:"BlkioWeight"`
				BlkioWeightDevice    interface{} `json:"BlkioWeightDevice"`
				BlkioDeviceReadBps   interface{} `json:"BlkioDeviceReadBps"`
				BlkioDeviceWriteBps  interface{} `json:"BlkioDeviceWriteBps"`
				BlkioDeviceReadIOps  interface{} `json:"BlkioDeviceReadIOps"`
				BlkioDeviceWriteIOps interface{} `json:"BlkioDeviceWriteIOps"`
				CPUPeriod            int         `json:"CpuPeriod"`
				CPUQuota             int         `json:"CpuQuota"`
				CpusetCpus           string      `json:"CpusetCpus"`
				CpusetMems           string      `json:"CpusetMems"`
				Devices              interface{} `json:"Devices"`
				DiskQuota            int         `json:"DiskQuota"`
				KernelMemory         int         `json:"KernelMemory"`
				MemoryReservation    int         `json:"MemoryReservation"`
				MemorySwap           int         `json:"MemorySwap"`
				MemorySwappiness     int         `json:"MemorySwappiness"`
				OomKillDisable       bool        `json:"OomKillDisable"`
				PidsLimit            int         `json:"PidsLimit"`
				Ulimits              interface{} `json:"Ulimits"`
				CPUCount             int         `json:"CpuCount"`
				CPUPercent           int         `json:"CpuPercent"`
				IOMaximumIOps        int         `json:"IOMaximumIOps"`
				IOMaximumBandwidth   int         `json:"IOMaximumBandwidth"`
			} `json:"HostConfig"`
			GraphDriver struct {
				Name string      `json:"Name"`
				Data interface{} `json:"Data"`
			} `json:"GraphDriver"`
			Mounts []struct {
				Source      string `json:"Source"`
				Destination string `json:"Destination"`
				Mode        string `json:"Mode"`
				RW          bool   `json:"RW"`
				Propagation string `json:"Propagation"`
			} `json:"Mounts"`
			Config struct {
				Hostname     string `json:"Hostname"`
				Domainname   string `json:"Domainname"`
				User         string `json:"User"`
				AttachStdin  bool   `json:"AttachStdin"`
				AttachStdout bool   `json:"AttachStdout"`
				AttachStderr bool   `json:"AttachStderr"`
				ExposedPorts struct {
					Four500UDP struct {
					} `json:"4500/udp"`
					Five00UDP struct {
					} `json:"500/udp"`
				} `json:"ExposedPorts"`
				Tty       bool     `json:"Tty"`
				OpenStdin bool     `json:"OpenStdin"`
				StdinOnce bool     `json:"StdinOnce"`
				Env       []string `json:"Env"`
				Cmd       []string `json:"Cmd"`
				Image     string   `json:"Image"`
				Volumes   struct {
					VarLibRancherEtc struct {
					} `json:"/var/lib/rancher/etc"`
				} `json:"Volumes"`
				WorkingDir string      `json:"WorkingDir"`
				Entrypoint interface{} `json:"Entrypoint"`
				MacAddress string      `json:"MacAddress"`
				OnBuild    interface{} `json:"OnBuild"`
				Labels     struct {
					IoRancherContainerAgentID string `json:"io.rancher.container.agent_id"`
					IoRancherContainerIP      string `json:"io.rancher.container.ip"`
					IoRancherContainerName    string `json:"io.rancher.container.name"`
					IoRancherContainerSystem  string `json:"io.rancher.container.system"`
					IoRancherContainerUUID    string `json:"io.rancher.container.uuid"`
				} `json:"Labels"`
			} `json:"Config"`
			NetworkSettings struct {
				Bridge                 string `json:"Bridge"`
				SandboxID              string `json:"SandboxID"`
				HairpinMode            bool   `json:"HairpinMode"`
				LinkLocalIPv6Address   string `json:"LinkLocalIPv6Address"`
				LinkLocalIPv6PrefixLen int    `json:"LinkLocalIPv6PrefixLen"`
				Ports                  struct {
					Four500UDP []struct {
						HostIP   string `json:"HostIp"`
						HostPort string `json:"HostPort"`
					} `json:"4500/udp"`
					Five00UDP []struct {
						HostIP   string `json:"HostIp"`
						HostPort string `json:"HostPort"`
					} `json:"500/udp"`
				} `json:"Ports"`
				SandboxKey             string      `json:"SandboxKey"`
				SecondaryIPAddresses   interface{} `json:"SecondaryIPAddresses"`
				SecondaryIPv6Addresses interface{} `json:"SecondaryIPv6Addresses"`
				EndpointID             string      `json:"EndpointID"`
				Gateway                string      `json:"Gateway"`
				GlobalIPv6Address      string      `json:"GlobalIPv6Address"`
				GlobalIPv6PrefixLen    int         `json:"GlobalIPv6PrefixLen"`
				IPAddress              string      `json:"IPAddress"`
				IPPrefixLen            int         `json:"IPPrefixLen"`
				IPv6Gateway            string      `json:"IPv6Gateway"`
				MacAddress             string      `json:"MacAddress"`
				Networks               struct {
					Bridge struct {
						IPAMConfig          interface{} `json:"IPAMConfig"`
						Links               interface{} `json:"Links"`
						Aliases             interface{} `json:"Aliases"`
						NetworkID           string      `json:"NetworkID"`
						EndpointID          string      `json:"EndpointID"`
						Gateway             string      `json:"Gateway"`
						IPAddress           string      `json:"IPAddress"`
						IPPrefixLen         int         `json:"IPPrefixLen"`
						IPv6Gateway         string      `json:"IPv6Gateway"`
						GlobalIPv6Address   string      `json:"GlobalIPv6Address"`
						GlobalIPv6PrefixLen int         `json:"GlobalIPv6PrefixLen"`
						MacAddress          string      `json:"MacAddress"`
					} `json:"bridge"`
				} `json:"Networks"`
			} `json:"NetworkSettings"`
		} `json:"dockerInspect"`
		DockerMounts []struct {
			Source      string `json:"Source"`
			Destination string `json:"Destination"`
			Mode        string `json:"Mode"`
			RW          bool   `json:"RW"`
			Propagation string `json:"Propagation"`
		} `json:"dockerMounts"`
	} `json:"data"`
	Token              string      `json:"token"`
	StartCount         int         `json:"startCount"`
	SystemContainer    string      `json:"systemContainer"`
	Removed            interface{} `json:"removed"`
	Userdata           interface{} `json:"userdata"`
	MemoryMb           interface{} `json:"memoryMb"`
	ServiceIndexID     interface{} `json:"serviceIndexId"`
	RemoveTime         interface{} `json:"removeTime"`
	AllocationState    string      `json:"allocationState"`
	Compute            interface{} `json:"compute"`
	NativeContainer    bool        `json:"nativeContainer"`
	NetworkContainerID interface{} `json:"networkContainerId"`
	DNSInternal        interface{} `json:"dnsInternal"`
	DNSSearchInternal  interface{} `json:"dnsSearchInternal"`
	Version            string      `json:"version"`
}
