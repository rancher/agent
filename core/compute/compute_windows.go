package compute

import (
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types/container"
	"github.com/rancher/agent/model"
)

func setupPublishPorts(hostConfig *container.HostConfig, instance model.Instance) {}

func setupDNSSearch(hostConfig *container.HostConfig, instance model.Instance) error {
	return nil
}

func setupLinks(hostConfig *container.HostConfig, instance model.Instance) {}

func setupNetworking(instance model.Instance, host model.Host, config *container.Config, hostConfig *container.HostConfig, client *client.Client) error {
	return nil
}

func setupFieldsHostConfig(fields model.InstanceFields, hostConfig *container.HostConfig) {

	hostConfig.LogConfig.Type = fields.LogConfig.Driver

	hostConfig.LogConfig.Config = fields.LogConfig.Config

	hostConfig.Isolation = fields.Isolation

	hostConfig.RestartPolicy = fields.RestartPolicy

	hostConfig.ConsoleSize = fields.ConsoleSize

	hostConfig.CPUCount = fields.CPUCount

	hostConfig.CPUPercent = fields.CPUPercent

	hostConfig.IOMaximumIOps = fields.IOMaximumIOps

	hostConfig.IOMaximumBandwidth = fields.IOMaximumBandwidth
}

func setupDeviceOptions(hostConfig *container.HostConfig, instance model.Instance, infoData model.InfoData) {
}

func setupComputeResourceFields(hostConfig *container.HostConfig, instance model.Instance) {

}
