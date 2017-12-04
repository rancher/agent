package compute

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/rancher/agent/model"
	dutils "github.com/rancher/agent/utilities/docker"
)

func setupPublishPorts(hostConfig *container.HostConfig, instance model.Instance) {}

func setupDNSSearch(hostConfig *container.HostConfig, instance model.Instance) error {
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

func setupDeviceOptions(hostConfig *container.HostConfig, instance model.Instance, infoData model.InfoData) {
}

func setupComputeResourceFields(hostConfig *container.HostConfig, instance model.Instance) {

}

var cmdDNS = []string{
	"powershell",
	"Get-NetAdapter | Foreach { " +
		"$a = (Get-DnsClientServerAddress -InterfaceIndex $_.ifIndex -Addressfamily IPv4).ServerAddresses + '169.254.169.251'; " +
		"Set-DnsClientServerAddress -InterfaceIndex $_.ifIndex -ServerAddresses $a }",
}

func configureDNS(dockerClient *client.Client, containerID string) error {
	var err error
	var execObj types.ContainerExecCreateResponse
	execConfig := types.ExecConfig{
		AttachStdout: true,
		AttachStdin:  true,
		AttachStderr: true,
		Privileged:   true,
		Tty:          false,
		Detach:       false,
		Cmd:          cmdDNS,
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
