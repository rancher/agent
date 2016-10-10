// +build linux freebsd solaris openbsd darwin

package compute

import (
	"bufio"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types/blkiodev"
	"github.com/docker/engine-api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/pkg/errors"
	"github.com/rancher/agent/core/hostInfo"
	"github.com/rancher/agent/model"
	"github.com/rancher/agent/utilities/constants"
	"github.com/rancher/agent/utilities/utils"
	"os"
	"strconv"
	"strings"
)

func setupPublishPorts(hostConfig *container.HostConfig, instance model.Instance) {
	hostConfig.PublishAllPorts = instance.Data.Fields.PublishAllPorts
}

func setupDNSSearch(hostConfig *container.HostConfig, instance model.Instance) error {
	systemCon := instance.SystemContainer
	if systemCon != "" {
		return nil
	}
	// if only rancher search is specified,
	// prepend search with params read from the system
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
			s = strings.Split(line, " ")[1:]
			for i := range s {
				search := s[len(s)-i-1]
				if !utils.SearchInList(s, search) {
					dnsSearch = append(dnsSearch, search)
				}
			}
			hostConfig.DNSSearch = dnsSearch
		}
	}
	return nil
}

func setupLinks(hostConfig *container.HostConfig, instance model.Instance) {
	links := []string{}

	if instance.InstanceLinks == nil {
		return
	}
	for _, link := range instance.InstanceLinks {
		if link.TargetInstance.UUID != "" {
			linkStr := fmt.Sprintf("%s:%s", link.TargetInstance.UUID, link.LinkName)
			links = append(links, linkStr)
		}
	}
	hostConfig.Links = links
}

func setupNetworking(instance model.Instance, host model.Host, config *container.Config, hostConfig *container.HostConfig, client *client.Client) error {
	portsSupported, hostnameSupported, err := setupNetworkMode(instance, client, config, hostConfig)
	if err != nil {
		return errors.Wrap(err, constants.SetupNetworkingError+"failed to setup network mode")
	}
	setupMacAndIP(instance, config, portsSupported, hostnameSupported)
	setupPortsNetwork(instance, config, hostConfig, portsSupported)
	setupLinksNetwork(instance, config, hostConfig)
	setupIpsec(instance, host, config, hostConfig)
	setupDNS(instance)
	return nil
}

func setupMacAndIP(instance model.Instance, config *container.Config, setMac bool, setHostname bool) {
	/*
		Configures the mac address and primary ip address for the the supplied
		container. The macAddress is configured directly as part of the native
		docker API. The primary IP address is set as an environment variable on the
		container. Another Rancher micro-service will detect this environment
		variable when the container is started and inject the IP into the
		container.

		Note: while an instance can technically have more than one nic based on the
		resource schema, this implementation assumes a single nic for the purpose
		of configuring the mac address and IP.
	*/
	macAddress := ""
	deviceNumber := -1
	for _, nic := range instance.Nics {
		if deviceNumber == -1 {
			macAddress = nic.MacAddress
			deviceNumber = nic.DeviceNumber
		} else if deviceNumber > nic.DeviceNumber {
			macAddress = nic.MacAddress
			deviceNumber = nic.DeviceNumber
		}
	}
	if setMac {
		config.MacAddress = macAddress
	}

	if !setHostname {
		config.Hostname = ""
	}

	if instance.Nics != nil && len(instance.Nics) > 0 && instance.Nics[0].IPAddresses != nil {
		// Assume one nic
		nic := instance.Nics[0]
		ipAddress := ""
		for _, ip := range nic.IPAddresses {
			if ip.Role == "primary" {
				ipAddress = fmt.Sprintf("%s/%s", ip.Address, strconv.Itoa(ip.Subnet.CidrSize))
				break
			}
		}
		if ipAddress != "" {
			utils.AddLabel(config, "io.rancher.container.ip", ipAddress)
		}
	}
}

func setupNetworkMode(instance model.Instance, client *client.Client,
	config *container.Config, hostConfig *container.HostConfig) (bool, bool, error) {
	/*
			Based on the network configuration we choose the network mode to set in
		    Docker.  We only really look for none, host, or container.  For all
		    all other configurations we assume bridge mode
	*/
	portsSupported := true
	hostnameSupported := true
	if len(instance.Nics) > 0 {
		kind := instance.Nics[0].Network.Kind
		if kind == "dockerHost" {
			portsSupported = false
			hostnameSupported = false
			config.NetworkDisabled = false
			hostConfig.NetworkMode = "host"
			hostConfig.Links = nil
		} else if kind == "dockerNone" {
			portsSupported = false
			config.NetworkDisabled = true
			hostConfig.NetworkMode = "none"
			hostConfig.Links = nil
		} else if kind == "dockerContainer" {
			portsSupported = false
			hostnameSupported = false
			id := instance.NetworkContainer.UUID
			other, err := utils.GetContainer(client, (*instance.NetworkContainer), false)
			if err != nil {
				if !utils.IsContainerNotFoundError(err) {
					return false, false, errors.Wrap(err, constants.SetupNetworkModeError+"failed to get container")
				}
			}
			if other.ID != "" {
				id = other.ID
			}
			hostConfig.NetworkMode = container.NetworkMode(fmt.Sprintf("container:%v", id))
			hostConfig.Links = nil
		}
	}
	return portsSupported, hostnameSupported, nil

}

func setupPortsNetwork(instance model.Instance, config *container.Config,
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

func setupIpsec(instance model.Instance, host model.Host, config *container.Config,
	hostConfig *container.HostConfig) {
	/*
			If the supplied instance is a network agent, configures the ports needed
		    to achieve multi-host networking.
	*/
	networkAgent := false
	if instance.SystemContainer == "" || instance.SystemContainer == "NetworkAgent" {
		networkAgent = true
	}
	if !networkAgent || !utils.HasService(instance, "ipsecTunnelService") {
		return
	}
	hostID := strconv.Itoa(host.ID)
	if endPoint, ok := instance.Data.IPSec[hostID]; ok {
		natValue := endPoint.Nat
		isakmp := endPoint.Isakmp
		binding := hostConfig.PortBindings

		port1 := nat.Port(fmt.Sprintf("%v/%v", 500, "udp"))
		port2 := nat.Port(fmt.Sprintf("%v/%v", 4500, "udp"))
		bind1 := nat.PortBinding{
			HostIP:   "0.0.0.0",
			HostPort: strconv.Itoa(int(isakmp)),
		}
		bind2 := nat.PortBinding{
			HostIP:   "0.0.0.0",
			HostPort: strconv.Itoa(int(natValue)),
		}
		exposedPorts := map[nat.Port]struct{}{
			port1: {},
			port2: {},
		}
		if _, ok := binding[port1]; ok {
			binding[port1] = append(binding[port1], bind1)
		} else {
			binding[port1] = []nat.PortBinding{bind1}
		}
		if _, ok := binding[port2]; ok {
			binding[port2] = append(binding[port2], bind1)
		} else {
			binding[port2] = []nat.PortBinding{bind2}
		}
		if len(config.ExposedPorts) > 0 {
			existingMap := config.ExposedPorts
			for port := range exposedPorts {
				existingMap[port] = struct{}{}
			}
			config.ExposedPorts = existingMap
		} else {
			config.ExposedPorts = exposedPorts
		}
	}
}

func setupDNS(instance model.Instance) {
	if !utils.HasService(instance, "dnsService") || instance.Kind == "virtualMachine" {
		return
	}
	ipAddress, macAddress, subnet := utils.FindIPAndMac(instance)

	if ipAddress == "" || macAddress == "" {
		return
	}

	parts := strings.Split(ipAddress, ".")
	if len(parts) != 4 {
		return
	}
	part2, _ := strconv.Atoi(parts[2])
	part3, _ := strconv.Atoi(parts[3])
	mark := strconv.Itoa(part2*1000 + part3)

	utils.CheckOutput([]string{"iptables", "-w", "-t", "nat", "-A", "CATTLE_PREROUTING",
		"!", "-s", subnet, "-d", "169.254.169.250", "-m", "mac",
		"--mac-source", macAddress, "-j", "MARK", "--set-mark",
		mark})
	utils.CheckOutput([]string{"iptables", "-w", "-t", "nat", "-A", "CATTLE_POSTROUTING",
		"!", "-s", subnet, "-d", "169.254.169.250", "-m", "mark", "--mark", mark,
		"-j", "SNAT", "--to", ipAddress})

}

func setupLinksNetwork(instance model.Instance, config *container.Config,
	hostConfig *container.HostConfig) {
	/*
			Sets up a container's config for rancher-managed links by removing the
		    docker native link configuration and emulating links through environment
		    variables.

		    Note that a non-rancher container (one created and started outside the
		    rancher API) container will not have its link configuration manipulated.
		    This is because on a container restart, we would not be able to properly
		    rebuild the link configuration because it depends on manipulating the
		    createConfig.
	*/
	if !utils.HasService(instance, "linkService") || utils.IsNonrancherContainer(instance) {
		return
	}

	hostConfig.Links = nil

	result := map[string]string{}
	if instance.InstanceLinks != nil {
		for _, link := range instance.InstanceLinks {
			linkName := link.LinkName
			utils.AddLinkEnv(linkName, link, result, "")
			utils.CopyLinkEnv(linkName, link, result)
			names := link.Data.Fields.InstanceNames
			for _, name := range names {
				utils.AddLinkEnv(name, link, result, linkName)
				utils.CopyLinkEnv(name, link, result)
				// This does assume the format {env}_{name}
				parts := strings.SplitN(name, "_", 2)
				if len(parts) == 1 {
					continue
				}
				utils.AddLinkEnv(parts[1], link, result, linkName)
				utils.CopyLinkEnv(parts[1], link, result)
			}
		}
		if len(result) > 0 {
			utils.AddToEnv(config, result)
		}
	}

}

// this method convert fields data to fields in host configuration
func setupFieldsHostConfig(fields model.InstanceFields, hostConfig *container.HostConfig) {

	hostConfig.ExtraHosts = fields.ExtraHosts

	hostConfig.PidMode = fields.PidMode

	hostConfig.LogConfig.Type = fields.LogConfig.Driver

	hostConfig.LogConfig.Config = fields.LogConfig.Config

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

	hostConfig.DNS = fields.DNS

	if hostConfig.DNSSearch == nil {
		hostConfig.DNSSearch = fields.DNSSearch
	}

	hostConfig.CapAdd = fields.CapAdd

	hostConfig.CapDrop = fields.CapDrop

	hostConfig.RestartPolicy = fields.RestartPolicy

	hostConfig.VolumeDriver = fields.VolumeDriver

	hostConfig.CpusetCpus = fields.CPUSet

	hostConfig.BlkioWeight = fields.BlkioWeight

	hostConfig.CgroupParent = fields.CgroupParent

	hostConfig.CPUPeriod = fields.CPUPeriod

	hostConfig.CPUQuota = fields.CPUQuota

	hostConfig.CpusetMems = fields.CPUsetMems

	hostConfig.DNSOptions = fields.DNSOpt

	hostConfig.GroupAdd = fields.GroupAdd

	hostConfig.KernelMemory = fields.KernelMemory

	hostConfig.MemorySwap = fields.MemorySwap

	hostConfig.Memory = fields.Memory

	hostConfig.MemorySwappiness = fields.MemorySwappiness

	hostConfig.OomKillDisable = fields.OomKillDisable

	hostConfig.ShmSize = fields.ShmSize

	hostConfig.Tmpfs = fields.Tmpfs

	hostConfig.Ulimits = fields.Ulimits

	hostConfig.UTSMode = fields.Uts

	hostConfig.IpcMode = fields.IpcMode
}

func setupComputeResourceFields(hostConfig *container.HostConfig, instance model.Instance) {
	hostConfig.MemoryReservation = instance.MemoryReservation

	shares := instance.Data.Fields.CPUShares
	if instance.MilliCPUReservation != 0 {
		// Need to do it this way instead of (milliCPU / milliCPUToCPU) * sharesPerCPU to avoid integer division resulting in a zero
		shares = (instance.MilliCPUReservation * 1024) / 1000
	}

	// Min value in kernel is 2
	if shares < 2 {
		shares = 2
	}

	hostConfig.CPUShares = shares
}

func setupDeviceOptions(hostConfig *container.HostConfig, instance model.Instance, infoData model.InfoData) {
	deviceOptions := instance.Data.Fields.BlkioDeviceOptions

	blkioWeightDevice := []*blkiodev.WeightDevice{}
	blkioDeviceReadIOps := []*blkiodev.ThrottleDevice{}
	blkioDeviceWriteBps := []*blkiodev.ThrottleDevice{}
	blkioDeviceReadBps := []*blkiodev.ThrottleDevice{}
	blkioDeviceWriteIOps := []*blkiodev.ThrottleDevice{}

	for dev, options := range deviceOptions {
		if dev == "DEFAULT_DISK" {
			// ignore this error because if we can't find the device we just skip that device
			dev, _ = hostInfo.GetDefaultDisk(infoData)
			if dev == "" {
				logrus.Warn(fmt.Sprintf("Couldn't find default device. Not setting device options: %s", options))
				continue
			}
		}
		if options.Weight != 0 {
			blkioWeightDevice = append(blkioWeightDevice, &blkiodev.WeightDevice{
				Path:   dev,
				Weight: options.Weight,
			})
		}
		if options.ReadIops != 0 {
			blkioDeviceReadIOps = append(blkioDeviceReadIOps, &blkiodev.ThrottleDevice{
				Path: dev,
				Rate: options.ReadIops,
			})
		}
		if options.WriteIops != 0 {
			blkioDeviceWriteIOps = append(blkioDeviceWriteIOps, &blkiodev.ThrottleDevice{
				Path: dev,
				Rate: options.WriteIops,
			})
		}
		if options.ReadBps != 0 {
			blkioDeviceReadBps = append(blkioDeviceReadBps, &blkiodev.ThrottleDevice{
				Path: dev,
				Rate: options.ReadBps,
			})
		}
		if options.WriteBps != 0 {
			blkioDeviceWriteBps = append(blkioDeviceWriteBps, &blkiodev.ThrottleDevice{
				Path: dev,
				Rate: options.WriteBps,
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

}
