package compute

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/rancher/agent/model"
)

func setupPublishPorts(hostConfig *container.HostConfig, instance model.Instance) {}

func setupDNSSearch(hostConfig *container.HostConfig, instance model.Instance) error {
	return nil
}

func setupLinks(hostConfig *container.HostConfig, instance model.Instance) {}

func setupNetworking(instance model.Instance, host model.Host, config *container.Config, hostConfig *container.HostConfig, client *client.Client, infoData model.InfoData) error {
	if len(instance.Nics) > 0 {
		hostConfig.NetworkMode = container.NetworkMode(instance.Nics[0].Network.Kind)
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

	hostConfig.SecurityOpt = fields.SecurityOpt
}

func setupDeviceOptions(hostConfig *container.HostConfig, instance model.Instance, infoData model.InfoData) {
}

func setupComputeResourceFields(hostConfig *container.HostConfig, instance model.Instance) {

}

func dockerContainerCreate(ctx context.Context, dockerClient *client.Client, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, containerName string) (types.ContainerCreateResponse, error) {
	return dockerClient.ContainerCreate(context.Background(), config, hostConfig, networkingConfig, containerName)
}
