package compute

import (
	"context"
	"strings"

	"github.com/Sirupsen/logrus"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/rancher/agent/model"
	dutils "github.com/rancher/agent/utilities/docker"
)

const (
	RancherDNSPriority = "io.rancher.container.dns.priority"
	RancherDomain      = "rancher.internal"
)

func setupPublishPorts(hostConfig *container.HostConfig, instance model.Instance) {}

func setupDNSSearch(hostConfig *container.HostConfig, instance model.Instance) error {
	var defaultDomains []string
	var svcNameSpace string
	var stackNameSpace string

	if instance.Data.Fields.Labels != nil {
		setRancherSearchDomains := true
		if strings.EqualFold(strings.TrimSpace(instance.Data.Fields.Labels[RancherDNSPriority]), "None") {
			setRancherSearchDomains = false
		}
		if setRancherSearchDomains {
			if value, ok := instance.Data.Fields.Labels["io.rancher.stack_service.name"]; ok {
				splitted := strings.Split(value, "/")
				svc := strings.ToLower(splitted[1])
				stack := strings.ToLower(splitted[0])
				svcNameSpace = svc + "." + stack + "." + RancherDomain
				stackNameSpace = stack + "." + RancherDomain
				defaultDomains = append(defaultDomains, svcNameSpace)
				defaultDomains = append(defaultDomains, stackNameSpace)
			}
		}
	}
	hostConfig.DNSSearch = defaultDomains
	return nil
}

func setupLinks(hostConfig *container.HostConfig, instance model.Instance) {}

func setupNetworking(instance model.Instance, host model.Host, config *container.Config, hostConfig *container.HostConfig, client *client.Client, infoData model.InfoData) error {
	if len(instance.Nics) > 0 {
		hostConfig.NetworkMode = container.NetworkMode(instance.Nics[0].Network.Kind)
		switch instance.Nics[0].Network.Kind {
		case "transparent":
			hostConfig.PublishAllPorts = false
			config.ExposedPorts = map[nat.Port]struct{}{}
			hostConfig.PortBindings = nat.PortMap{}
		default:
		}
	}
	return nil
}

func setupFieldsHostConfig(fields model.InstanceFields, hostConfig *container.HostConfig) {

	hostConfig.LogConfig.Type = fields.LogConfig.Driver

	hostConfig.LogConfig.Config = fields.LogConfig.Config

	hostConfig.Isolation = fields.Isolation

	hostConfig.RestartPolicy = fields.RestartPolicy

	hostConfig.CPUCount = fields.CPUCount

	hostConfig.CPUPercent = fields.CPUPercent

	hostConfig.IOMaximumIOps = fields.IOMaximumIOps

	hostConfig.IOMaximumBandwidth = fields.IOMaximumBandwidth
}

func setupRancherFlexVolume(instance model.Instance, hostConfig *container.HostConfig) error {
	return nil
}

func setupDeviceOptions(hostConfig *container.HostConfig, instance model.Instance, infoData model.InfoData) {
}

func setupComputeResourceFields(hostConfig *container.HostConfig, instance model.Instance) {

}

var cmdDNS = []string{
	"powershell",
	"Get-NetAdapter | Foreach { " +
		"$a =@('169.254.169.251') + (Get-DnsClientServerAddress -InterfaceIndex $_.ifIndex -Addressfamily IPv4).ServerAddresses ; " +
		"Set-DnsClientServerAddress -InterfaceIndex $_.ifIndex -ServerAddresses $a }",
}

func configureDNS(dockerClient *client.Client, containerID string) error {
	var err error
	var execObj types.ContainerExecCreateResponse
	var currentCMD []string
	currentCMD = append(currentCMD, cmdDNS...)
	//Setup dns search list
	info, err := dockerClient.ContainerInspect(context.Background(), containerID)
	if err == nil && len(info.HostConfig.DNSSearch) != 0 {
		dnsSearch := []string{}
		for i := 0; i < len(info.HostConfig.DNSSearch); i++ {
			dnsSearch = append(dnsSearch, `"`+info.HostConfig.DNSSearch[i]+`"`)
		}
		currentCMD[1] = currentCMD[1] + "; Set-DnsClientGlobalSetting -SuffixSearchList " + strings.Join(dnsSearch, ",")
	}
	logrus.Info(currentCMD)
	execConfig := types.ExecConfig{
		AttachStdout: true,
		AttachStdin:  true,
		AttachStderr: true,
		Privileged:   true,
		Tty:          false,
		Detach:       false,
		Cmd:          currentCMD,
	}

	if execObj, err = dockerClient.ContainerExecCreate(context.Background(), containerID, execConfig); err == nil {
		err = dockerClient.ContainerExecStart(context.Background(), execObj.ID, types.ExecStartCheck{})
	}
	return err
}

func dockerContainerCreate(ctx context.Context, dockerClient *client.Client, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, containerName string) (types.ContainerCreateResponse, error) {
	var (
		ret types.ContainerCreateResponse
		err error
	)
	dutils.Serialize(func() error {
		ret, err = dockerClient.ContainerCreate(context.Background(), config, hostConfig, networkingConfig, containerName)
		return err
	})
	return ret, err
}

func unmountRancherFlexVolume(instance model.Instance) error {
	return nil
}
