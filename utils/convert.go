package utils

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/api/types"
	v3 "github.com/rancher/go-rancher/v3"
	"strconv"
	"strings"
)

const (
	readIops         = "readIops"
	writeIops        = "writeIops"
	readBps          = "readBps"
	writeBps         = "writeBps"
	weight           = "weight"
	networkNone      = "none"
	networkBridge    = "bridge"
	networkHost      = "host"
	networkContainer = "container"
)

//ConvertInspect converts a docker inspect into a rancher container api, can be moved to a common library that does conversion
func ConvertInspect(inspect types.ContainerJSON, original v3.Container) v3.Container {
	result := original
	result.ExternalId = inspect.ID
	result.BlkioDeviceOptions = convertBlkioOptions(inspect)
	result.BlkioWeight = int64(inspect.HostConfig.BlkioWeight)
	result.CapAdd = inspect.HostConfig.CapAdd
	result.CapDrop = inspect.HostConfig.CapDrop
	result.CgroupParent = inspect.HostConfig.CgroupParent
	result.Command = inspect.Config.Cmd
	result.CpuCount = inspect.HostConfig.CPUCount
	result.CpuPercent = inspect.HostConfig.CPUPercent
	result.CpuPeriod = inspect.HostConfig.CPUPeriod
	result.CpuQuota = inspect.HostConfig.CPUQuota
	result.CpuSet = inspect.HostConfig.CpusetCpus
	result.CpuSetMems = inspect.HostConfig.CpusetMems
	result.CpuShares = inspect.HostConfig.CPUShares
	result.Devices = convertDevice(inspect)
	result.DiskQuota = inspect.HostConfig.DiskQuota
	result.Dns = inspect.HostConfig.DNS
	result.DnsOpt = inspect.HostConfig.DNSOptions
	result.DnsSearch = inspect.HostConfig.DNSSearch
	result.DomainName = inspect.Config.Domainname
	result.EntryPoint = inspect.Config.Entrypoint
	result.Environment = convertEnv(inspect)
	result.ExitCode = int64(inspect.State.ExitCode)
	result.Expose = convertExposed(inspect)
	result.ExtraHosts = inspect.HostConfig.ExtraHosts
	result.GroupAdd = inspect.HostConfig.GroupAdd
	if inspect.Config.Healthcheck != nil {
		result.HealthCmd = inspect.Config.Healthcheck.Test
		result.HealthInterval = int64(inspect.Config.Healthcheck.Interval.Seconds())
		result.HealthRetries = int64(inspect.Config.Healthcheck.Retries)
		result.HealthTimeout = int64(inspect.Config.Healthcheck.Timeout.Seconds())
	}
	result.Hostname = inspect.Config.Hostname
	result.IoMaximumBandwidth = int64(inspect.HostConfig.IOMaximumBandwidth)
	result.IoMaximumIOps = int64(inspect.HostConfig.IOMaximumIOps)
	result.IpcMode = string(inspect.HostConfig.IpcMode)
	result.Isolation = string(inspect.HostConfig.Isolation)
	result.KernelMemory = inspect.HostConfig.KernelMemory
	result.Labels = toMapInterface(inspect.Config.Labels)
	if inspect.HostConfig.LogConfig.Type != "" {
		result.LogConfig = &v3.LogConfig{}
		result.LogConfig.Driver = inspect.HostConfig.LogConfig.Type
		result.LogConfig.Config = toMapInterface(inspect.HostConfig.LogConfig.Config)
	}
	result.Memory = inspect.HostConfig.Memory
	result.MemoryReservation = inspect.HostConfig.MemoryReservation
	result.MemorySwap = inspect.HostConfig.MemorySwap
	result.MemorySwappiness = *inspect.HostConfig.MemorySwappiness
	result.Name = strings.TrimPrefix(inspect.Name, "/")
	result.NetworkMode = convertNetworkMode(inspect)
	result.OomKillDisable = *inspect.HostConfig.OomKillDisable
	result.OomScoreAdj = int64(inspect.HostConfig.OomScoreAdj)
	result.PidMode = string(inspect.HostConfig.PidMode)
	result.PidsLimit = inspect.HostConfig.PidsLimit
	result.Privileged = inspect.HostConfig.Privileged
	result.PublishAllPorts = inspect.HostConfig.PublishAllPorts
	result.ReadOnly = inspect.HostConfig.ReadonlyRootfs
	result.SecurityOpt = inspect.HostConfig.SecurityOpt
	result.ShmSize = inspect.HostConfig.ShmSize
	result.StdinOpen = inspect.Config.OpenStdin
	result.StopSignal = inspect.Config.StopSignal
	result.StorageOpt = toMapInterface(inspect.HostConfig.StorageOpt)
	result.Sysctls = toMapInterface(inspect.HostConfig.Sysctls)
	result.Tmpfs = toMapInterface(inspect.HostConfig.Tmpfs)
	result.Tty = inspect.Config.Tty
	result.User = inspect.Config.User
	result.UsernsMode = string(inspect.HostConfig.UsernsMode)
	result.Uts = string(inspect.HostConfig.UTSMode)
	result.VolumeDriver = inspect.HostConfig.VolumeDriver
	result.WorkingDir = inspect.Config.WorkingDir
	result.Ulimits = convertUlimit(inspect)
	result.PublicEndpoints = convertPublicEndpoint(inspect)
	result.Ports = convertPortField(inspect)
	result.MountPoint = convertMountPoint(inspect)
	result.Image = inspect.Config.Image

	return result
}

type port struct {
	hostIP   string
	hostPort string
	port     string
	protocol string
}

func (p port) String() string {
	if p.hostIP == "" {
		if p.protocol == "" {
			return fmt.Sprintf("%s:%s/%s", p.hostPort, p.port, "tcp")
		}
		return fmt.Sprintf("%s:%s/%s", p.hostPort, p.port, p.protocol)
	}
	return fmt.Sprintf("%s:%s:%s/%s", p.hostIP, p.hostPort, p.port, p.protocol)
}

func convertMountPoint(inspect types.ContainerJSON) []v3.MountPoint {
	r := []v3.MountPoint{}
	for _, mount := range inspect.Mounts {
		r = append(r, v3.MountPoint{
			Name:        mount.Name,
			Driver:      mount.Driver,
			Destination: mount.Destination,
			Source:      mount.Source,
			Propagation: string(mount.Propagation),
			MountType:   string(mount.Type),
			Rw:          mount.RW,
		})
	}
	return r
}

func convertPort(inspect types.ContainerJSON) []port {
	r := []port{}
	for p, bindings := range inspect.HostConfig.PortBindings {
		for _, binding := range bindings {
			r = append(r, port{
				hostIP:   binding.HostIP,
				hostPort: binding.HostPort,
				port:     p.Port(),
				protocol: p.Proto(),
			})
		}
	}
	return r
}

func convertPublicEndpoint(inspect types.ContainerJSON) []v3.PublicEndpoint {
	r := []v3.PublicEndpoint{}
	for port, bindings := range inspect.HostConfig.PortBindings {
		for _, binding := range bindings {
			hostPort, err := strconv.Atoi(binding.HostPort)
			if err != nil {
				logrus.Errorf("can't convert hostPort field, err: %v", err)
				continue
			}
			r = append(r, v3.PublicEndpoint{
				Protocol:    port.Proto(),
				IpAddress:   binding.HostIP,
				PrivatePort: int64(port.Int()),
				PublicPort:  int64(hostPort),
			})
		}
	}
	return r
}

func convertPortField(inspect types.ContainerJSON) []string {
	r := []string{}
	for _, port := range convertPort(inspect) {
		r = append(r, port.String())
	}
	return r
}

func convertUlimit(inspect types.ContainerJSON) []v3.Ulimit {
	r := []v3.Ulimit{}
	for _, ul := range inspect.HostConfig.Ulimits {
		r = append(r, v3.Ulimit{
			Name: ul.Name,
			Soft: ul.Soft,
			Hard: ul.Hard,
		})
	}
	return r
}

func convertNetworkMode(inspect types.ContainerJSON) string {
	if inspect.Config.NetworkDisabled {
		return networkNone
	}
	mode := inspect.HostConfig.NetworkMode.NetworkName()
	if mode == networkHost || mode == networkBridge || mode == networkNone || strings.HasPrefix(mode, networkContainer) {
		return mode
	}
	if mode == "default" {
		return networkBridge
	}
	return ""
}

func convertExposed(inspect types.ContainerJSON) []string {
	r := []string{}
	for k := range inspect.Config.ExposedPorts {
		r = append(r, k.Port())
	}
	return r
}

func convertDevice(inspect types.ContainerJSON) []string {
	d := []string{}
	for _, device := range inspect.HostConfig.Devices {
		if device.CgroupPermissions != "" {
			d = append(d, fmt.Sprintf("%s:%s:%s", device.PathOnHost, device.PathInContainer, device.CgroupPermissions))
		} else {
			d = append(d, fmt.Sprintf("%s:%s:%s", device.PathOnHost, device.PathInContainer, "rwm"))
		}
	}
	return d
}

func convertEnv(inspect types.ContainerJSON) map[string]interface{} {
	r := map[string]interface{}{}
	for _, env := range inspect.Config.Env {
		parts := strings.Split(env, "=")
		if len(parts) > 1 {
			r[parts[0]] = parts[1]
		}
	}
	return r
}

func convertBlkioOptions(inspect types.ContainerJSON) map[string]interface{} {
	result := map[string]interface{}{}
	// BlkioWeightDevice
	for _, option := range inspect.HostConfig.BlkioWeightDevice {
		if v, ok := result[option.Path]; ok {
			v.(map[string]interface{})[weight] = int64(option.Weight)
		} else {
			result[option.Path] = map[string]interface{}{
				weight: int64(option.Weight),
			}
		}
	}
	// BlkioDeviceReadIOps
	for _, option := range inspect.HostConfig.BlkioDeviceReadIOps {
		if v, ok := result[option.Path]; ok {
			v.(map[string]interface{})[readIops] = int64(option.Rate)
		} else {
			result[option.Path] = map[string]interface{}{
				readIops: int64(option.Rate),
			}
		}
	}
	// BlkioDeviceWriteIOps
	for _, option := range inspect.HostConfig.BlkioDeviceWriteIOps {
		if v, ok := result[option.Path]; ok {
			v.(map[string]interface{})[writeIops] = int64(option.Rate)
		} else {
			result[option.Path] = map[string]interface{}{
				writeIops: int64(option.Rate),
			}
		}
	}
	// BlkioDeviceReadBps
	for _, option := range inspect.HostConfig.BlkioDeviceReadBps {
		if v, ok := result[option.Path]; ok {
			v.(map[string]interface{})[readBps] = int64(option.Rate)
		} else {
			result[option.Path] = map[string]interface{}{
				readBps: int64(option.Rate),
			}
		}
	}
	// BlkioDeviceWriteBps
	for _, option := range inspect.HostConfig.BlkioDeviceWriteBps {
		if v, ok := result[option.Path]; ok {
			v.(map[string]interface{})[writeBps] = int64(option.Rate)
		} else {
			result[option.Path] = map[string]interface{}{
				writeBps: int64(option.Rate),
			}
		}
	}
	return result
}
