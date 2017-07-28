// +build linux freebsd solaris openbsd darwin

package runtime

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/api/types/blkiodev"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/docker/go-units"
	"github.com/pkg/errors"
	"github.com/rancher/agent/host_info"
	"github.com/rancher/agent/progress"
	"github.com/rancher/agent/utils"
	v2 "github.com/rancher/go-rancher/v2"
)

const (
	rancherDNSPriority = "io.rancher.container.dns.priority"
)

func setupPublishPorts(hostConfig *container.HostConfig, containerSpec v2.Container) {
	hostConfig.PublishAllPorts = containerSpec.PublishAllPorts
}

func setupDNSSearch(hostConfig *container.HostConfig, containerSpec v2.Container) error {
	// if only rancher search is specified,
	// prepend search with params read from the system
	last := utils.InterfaceToString(containerSpec.Labels[rancherDNSPriority]) == "service_last"
	allRancher := true
	dnsSearch := hostConfig.DNSSearch

	if len(dnsSearch) == 0 {
		return nil
	}
	for _, search := range dnsSearch {
		if strings.HasSuffix(search, "rancher.internal") {
			continue
		}
		allRancher = false
		break
	}

	if !allRancher {
		return nil
	}

	// read host's resolv.conf
	file, err := os.Open("/etc/resolv.conf")
	if err != nil {
		return errors.Wrap(err, "Failed to set DNS search")
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		s := []string{}
		if strings.HasPrefix(line, "search") {
			// in case multiple search lines
			// respect the last one
			s = strings.Fields(line)[1:]
			for i := range s {
				var search string
				if last {
					search = s[len(s)-i-1]
				} else {
					search = s[i]
				}
				if !utils.SearchInList(dnsSearch, search) {
					if last {
						dnsSearch = append([]string{search}, dnsSearch...)
					} else {
						dnsSearch = append(dnsSearch, []string{search}...)
					}
				}
			}
			hostConfig.DNSSearch = dnsSearch
		}
	}
	return nil
}

func setupNetworking(containerSpec v2.Container, config *container.Config, hostConfig *container.HostConfig, idsMap map[string]string, networks []v2.Network) error {
	portsSupported, hostnameSupported, err := setupNetworkMode(containerSpec, networks, config, hostConfig, idsMap)
	if err != nil {
		return errors.Wrap(err, "failed to setup network mode")
	}

	if !hostnameSupported {
		config.Hostname = ""
	}
	setupPortsNetwork(config, hostConfig, portsSupported)
	return nil
}

func setupNetworkMode(containerSpec v2.Container, networks []v2.Network, config *container.Config, hostConfig *container.HostConfig, idsMap map[string]string) (bool, bool, error) {
	/*
			Based on the network configuration we choose the network mode to set in
		    Docker.  We only really look for none, host, or container.  For all
		    all other configurations we assume bridge mode
	*/
	portsSupported := true
	hostnameSupported := true
	kind := ""
	for _, network := range networks {
		if containerSpec.PrimaryNetworkId == network.Id {
			kind = network.Kind
		}
	}
	if kind == "dockerHost" {
		portsSupported = false
		hostnameSupported = false
		config.NetworkDisabled = false
		hostConfig.NetworkMode = "host"
		hostConfig.Links = nil
		dockerVersion := hostInfo.DockerData.Version.Version
		if strings.HasPrefix(dockerVersion, "1.10") || strings.HasPrefix(dockerVersion, "1.11") {
			hostConfig.DNS = nil
			hostConfig.DNSSearch = nil
		}
	} else if kind == "dockerNone" {
		portsSupported = false
		config.NetworkDisabled = true
		hostConfig.NetworkMode = "none"
		hostConfig.Links = nil
	} else if kind == "dockerContainer" {
		// TODO: find network container id
		portsSupported = false
		hostnameSupported = false
		if containerSpec.NetworkContainerId != "" && idsMap[containerSpec.NetworkContainerId] != "" {
			hostConfig.NetworkMode = container.NetworkMode(fmt.Sprintf("container:%v", idsMap[containerSpec.NetworkContainerId]))
			hostConfig.Links = nil
		}
	} else if kind == "cni" {
		portsSupported = false
		// If this is set true resolv.conf is not setup.
		config.NetworkDisabled = false
		hostConfig.NetworkMode = "none"
		hostConfig.Links = nil
		if strings.HasPrefix(hostInfo.DockerData.Version.Version, "1.10") {
			hostConfig.DNS = nil
			hostConfig.DNSSearch = nil
		}
	}
	return portsSupported, hostnameSupported, nil

}

func setupPortsNetwork(config *container.Config,
	hostConfig *container.HostConfig, portsSupported bool) {
	/*
			Docker 1.9+ does not allow you to pass port info for networks that don't
		    support ports (net, none, container:x)
	*/
	if !portsSupported {
		hostConfig.PublishAllPorts = false
		config.ExposedPorts = map[nat.Port]struct{}{}
		hostConfig.PortBindings = nat.PortMap{}
	}
}

// this method convert fields data to fields in host configuration
func setupFieldsHostConfig(fields v2.Container, hostConfig *container.HostConfig) {

	hostConfig.ExtraHosts = fields.ExtraHosts

	hostConfig.PidMode = container.PidMode(fields.PidMode)

	if fields.LogConfig != nil {
		hostConfig.LogConfig.Type = fields.LogConfig.Driver
		hostConfig.LogConfig.Config = utils.ToMapString(fields.LogConfig.Config)
	}

	hostConfig.SecurityOpt = fields.SecurityOpt

	deviceMappings := []container.DeviceMapping{}
	devices := fields.Devices
	for _, device := range devices {
		parts := strings.Split(device, ":")
		permission := "rwm"
		if len(parts) == 3 {
			permission = parts[2]
		}
		deviceMappings = append(deviceMappings,
			container.DeviceMapping{
				PathOnHost:        parts[0],
				PathInContainer:   parts[1],
				CgroupPermissions: permission,
			})
	}

	hostConfig.Devices = deviceMappings

	hostConfig.DNS = fields.Dns

	hostConfig.DNSSearch = fields.DnsSearch

	hostConfig.CapAdd = fields.CapAdd

	hostConfig.CapDrop = fields.CapDrop

	if fields.RestartPolicy != nil {
		hostConfig.RestartPolicy = container.RestartPolicy{
			Name:              fields.RestartPolicy.Name,
			MaximumRetryCount: int(fields.RestartPolicy.MaximumRetryCount),
		}
	}

	hostConfig.CpusetCpus = fields.CpuSet

	hostConfig.BlkioWeight = uint16(fields.BlkioWeight)

	hostConfig.CgroupParent = fields.CgroupParent

	hostConfig.CPUPeriod = fields.CpuPeriod

	hostConfig.CPUQuota = fields.CpuQuota

	hostConfig.CpusetMems = fields.CpuSetMems

	hostConfig.DNSOptions = fields.DnsOpt

	hostConfig.GroupAdd = fields.GroupAdd

	hostConfig.KernelMemory = fields.KernelMemory

	hostConfig.MemorySwap = fields.MemorySwap

	hostConfig.Memory = fields.Memory

	hostConfig.MemorySwappiness = &fields.MemorySwappiness

	hostConfig.OomKillDisable = &fields.OomKillDisable

	hostConfig.OomScoreAdj = int(fields.OomScoreAdj)

	hostConfig.ShmSize = fields.ShmSize

	hostConfig.Tmpfs = utils.ToMapString(fields.Tmpfs)

	hostConfig.Ulimits = convertUlimits(fields.Ulimits)

	hostConfig.UTSMode = container.UTSMode(fields.Uts)

	hostConfig.IpcMode = container.IpcMode(fields.IpcMode)

	hostConfig.Sysctls = utils.ToMapString(fields.Sysctls)

	hostConfig.StorageOpt = utils.ToMapString(fields.StorageOpt)

	hostConfig.PidsLimit = fields.PidsLimit

	hostConfig.DiskQuota = fields.DiskQuota

	hostConfig.CgroupParent = fields.CgroupParent

	hostConfig.UsernsMode = container.UsernsMode(fields.UsernsMode)
}

func convertUlimits(ulimits []v2.Ulimit) []*units.Ulimit {
	r := []*units.Ulimit{}
	for _, ulimit := range ulimits {
		r = append(r, &units.Ulimit{
			Hard: ulimit.Hard,
			Soft: ulimit.Soft,
			Name: ulimit.Name,
		})
	}
	return r
}

func setupComputeResourceFields(hostConfig *container.HostConfig, containerSpec v2.Container) {
	hostConfig.MemoryReservation = containerSpec.MemoryReservation

	shares := containerSpec.CpuShares
	if containerSpec.MilliCpuReservation != 0 {
		// Need to do it this way instead of (milliCPU / milliCPUToCPU) * sharesPerCPU to avoid integer division resulting in a zero
		shares = (containerSpec.MilliCpuReservation * 1024) / 1000
	}

	// Min value in kernel is 2
	if shares < 2 {
		shares = 2
	}

	hostConfig.CPUShares = shares
}

type deviceOptions struct {
	Weight    uint16
	ReadIops  uint64
	WriteIops uint64
	ReadBps   uint64
	WriteBps  uint64
}

func setupDeviceOptions(hostConfig *container.HostConfig, spec v2.Container) error {
	devOptions := spec.BlkioDeviceOptions

	blkioWeightDevice := []*blkiodev.WeightDevice{}
	blkioDeviceReadIOps := []*blkiodev.ThrottleDevice{}
	blkioDeviceWriteBps := []*blkiodev.ThrottleDevice{}
	blkioDeviceReadBps := []*blkiodev.ThrottleDevice{}
	blkioDeviceWriteIOps := []*blkiodev.ThrottleDevice{}

	for dev, value := range devOptions {
		if dev == "DEFAULT_DISK" {
			// ignore this error because if we can't find the device we just skip that device
			dev, _ = hostInfo.GetDefaultDisk()
			if dev == "" {
				logrus.Warn(fmt.Sprintf("Couldn't find default device. Not setting device options: %v", value))
				continue
			}
		}
		option := deviceOptions{}
		if err := utils.Unmarshalling(value, &option); err != nil {
			return err
		}
		if option.Weight != 0 {
			blkioWeightDevice = append(blkioWeightDevice, &blkiodev.WeightDevice{
				Path:   dev,
				Weight: option.Weight,
			})
		}
		if option.ReadIops != 0 {
			blkioDeviceReadIOps = append(blkioDeviceReadIOps, &blkiodev.ThrottleDevice{
				Path: dev,
				Rate: option.ReadIops,
			})
		}
		if option.WriteIops != 0 {
			blkioDeviceWriteIOps = append(blkioDeviceWriteIOps, &blkiodev.ThrottleDevice{
				Path: dev,
				Rate: option.WriteIops,
			})
		}
		if option.ReadBps != 0 {
			blkioDeviceReadBps = append(blkioDeviceReadBps, &blkiodev.ThrottleDevice{
				Path: dev,
				Rate: option.ReadBps,
			})
		}
		if option.WriteBps != 0 {
			blkioDeviceWriteBps = append(blkioDeviceWriteBps, &blkiodev.ThrottleDevice{
				Path: dev,
				Rate: option.WriteBps,
			})
		}
	}
	if len(blkioWeightDevice) > 0 {
		hostConfig.BlkioWeightDevice = blkioWeightDevice
	}
	if len(blkioDeviceReadIOps) > 0 {
		hostConfig.BlkioDeviceReadIOps = blkioDeviceReadIOps
	}
	if len(blkioDeviceWriteIOps) > 0 {
		hostConfig.BlkioDeviceWriteIOps = blkioDeviceWriteIOps
	}
	if len(blkioDeviceReadBps) > 0 {
		hostConfig.BlkioDeviceReadBps = blkioDeviceReadBps
	}
	if len(blkioDeviceWriteBps) > 0 {
		hostConfig.BlkioDeviceWriteBps = blkioDeviceWriteBps
	}
	return nil
}

func setupRancherFlexVolume(volumes []v2.Volume, dataVolumes []string, progress *progress.Progress) ([]string, error) {
	bindMounts := []string{}
	for _, volume := range volumes {
		if IsRancherVolume(volume) {
			payload := struct {
				Name    string
				Options map[string]string `json:"Opts,omitempty"`
			}{
				Name:    volume.Name,
				Options: utils.ToMapString(volume.DriverOpts),
			}
			progress.Update(fmt.Sprintf("Creating volume %s", volume.Name), "yes", nil)
			_, err := callRancherStorageVolumePlugin(volume, Create, payload)
			if err != nil {
				return nil, err
			}
			progress.Update(fmt.Sprintf("Attaching volume %s", volume.Name), "yes", nil)
			_, err = callRancherStorageVolumePlugin(volume, Attach, payload)
			if err != nil {
				return nil, err
			}
			progress.Update(fmt.Sprintf("Mounting volume %s", volume.Name), "yes", nil)
			resp, err := callRancherStorageVolumePlugin(volume, Mount, payload)
			if err != nil {
				return nil, err
			}
			for _, vol := range dataVolumes {
				parts := strings.Split(vol, ":")
				if len(parts) > 1 {
					mode := "rw"
					if len(parts) == 3 {
						mode = parts[2]
					}
					if volume.Name == parts[0] {
						bindMounts = append(bindMounts, fmt.Sprintf("%s:%s:%s", resp.Mountpoint, parts[1], mode))
					}
				}
			}

		}
	}
	return bindMounts, nil
}

func unmountRancherFlexVolume(volumes []v2.Volume) error {
	for _, volume := range volumes {
		if IsRancherVolume(volume) {
			payload := struct{ Name string }{Name: volume.Name}
			_, err := callRancherStorageVolumePlugin(volume, Unmount, payload)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func dockerContainerCreate(ctx context.Context, dockerClient *client.Client, config *container.Config, hostConfig *container.HostConfig, containerName string) (container.ContainerCreateCreatedBody, error) {
	var (
		ret container.ContainerCreateCreatedBody
		err error
	)
	if err := modifyForCNI(dockerClient, config, hostConfig); err != nil {
		return container.ContainerCreateCreatedBody{}, err
	}
	utils.Serialize(func() error {
		ret, err = dockerClient.ContainerCreate(context.Background(), config, hostConfig, nil, containerName)
		return err
	})
	return ret, err
}

func getContainerName(spec v2.Container) string {
	instanceName := spec.Name
	parts := strings.Split(spec.Uuid, "-")
	return fmt.Sprintf("r-%s-%s", instanceName, parts[0])
}
