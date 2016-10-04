package ping

import (
	"net"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/pkg/errors"
	"github.com/rancher/agent/core/hostInfo"
	"github.com/rancher/agent/model"
	"github.com/rancher/agent/utilities/config"
	"github.com/rancher/agent/utilities/constants"
	"github.com/rancher/agent/utilities/utils"
	revents "github.com/rancher/event-subscriber/events"
	"golang.org/x/net/context"
)

func addResource(ping *revents.Event, pong *model.PingResponse, dockerClient *client.Client, collectors []hostInfo.Collector) error {
	if !pingIncludeResource(ping) {
		return nil
	}
	stats := map[string]interface{}{}
	if pingIncludeStats(ping) {
		data := hostInfo.CollectData(collectors)
		stats = data
	}

	physicalHost, err := config.PhysicalHost()
	if err != nil {
		return errors.WithStack(err)
	}

	hostname, err := config.Hostname()
	if err != nil {
		return errors.WithStack(err)
	}
	labels, err := getHostLabels(collectors)
	if err != nil {
		logrus.Warnf("Failed to get Host Labels err msg: %v", err.Error())
	}
	rancherImage := os.Getenv("RANCHER_AGENT_IMAGE")
	labels[constants.RancherAgentImage] = rancherImage
	uuid, err := config.DockerUUID()
	if err != nil {
		return errors.WithStack(err)
	}
	compute := model.PingResource{
		Type:             "host",
		Kind:             "docker",
		HostName:         hostname,
		CreateLabels:     config.Labels(),
		Labels:           labels,
		UUID:             uuid,
		PhysicalHostUUID: physicalHost.UUID,
		Info:             stats,
		APIProxy:         config.HostProxy(),
	}

	pool := model.PingResource{
		Type:     "storagePool",
		Kind:     "docker",
		Name:     compute.HostName + " Storage Pool",
		HostUUID: compute.UUID,
		UUID:     compute.UUID + "-pool",
	}

	resolvedIP, err := net.LookupIP(config.DockerHostIP())
	ipAddr := ""
	if err != nil {
		logrus.Warn(err)
	} else {
		ipAddr = resolvedIP[0].String()
	}
	ip := model.PingResource{
		Type:     "ipAddress",
		UUID:     ipAddr,
		Addresss: ipAddr,
		HostUUID: compute.UUID,
	}
	pong.Resources = append(pong.Resources, physicalHost, compute, pool, ip)
	return nil
}

func addInstance(ping *revents.Event, pong *model.PingResponse, dockerClient *client.Client, systemImages map[string]string) error {
	if !pingIncludeInstance(ping) {
		return nil
	}
	uuid, err := config.DockerUUID()
	if err != nil {
		return errors.WithStack(err)
	}
	pong.Resources = append(pong.Resources, model.PingResource{
		Type: "hostUuid",
		UUID: uuid,
	})
	containers := []model.PingResource{}
	running, nonrunning, err := getAllContainerByState(dockerClient)
	if err != nil {
		return errors.WithStack(err)
	}
	for _, container := range running {
		containers = utils.AddContainer("running", container, containers, dockerClient, systemImages)
	}
	for _, container := range nonrunning {
		containers = utils.AddContainer("stopped", container, containers, dockerClient, systemImages)
	}
	pong.Resources = append(pong.Resources, containers...)
	pong.Options.Instances = true
	return nil
}

func pingIncludeResource(ping *revents.Event) bool {
	value, ok := utils.GetFieldsIfExist(ping.Data, "options", "resources")
	if !ok {
		return false
	}
	return utils.InterfaceToBool(value)
}

func pingIncludeStats(ping *revents.Event) bool {
	value, ok := utils.GetFieldsIfExist(ping.Data, "options", "stats")
	if !ok {
		return false
	}
	return utils.InterfaceToBool(value)
}

func pingIncludeInstance(ping *revents.Event) bool {
	value, ok := utils.GetFieldsIfExist(ping.Data, "options", "instances")
	if !ok {
		return false
	}
	return utils.InterfaceToBool(value)
}

func getHostLabels(collectors []hostInfo.Collector) (map[string]string, error) {
	return hostInfo.HostLabels("io.rancher.host", collectors)
}

func getAllContainerByState(dockerClient *client.Client) (map[string]types.Container, map[string]types.Container, error) {
	nonrunningContainers := map[string]types.Container{}
	containerList, err := dockerClient.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		return map[string]types.Container{}, map[string]types.Container{}, errors.WithStack(err)
	}
	for _, c := range containerList {
		if c.Status != "" && c.Status != "Created" {
			nonrunningContainers[c.ID] = c
		}
	}
	runningContainers := map[string]types.Container{}
	containerListRunning, _ := dockerClient.ContainerList(context.Background(), types.ContainerListOptions{})
	for _, c := range containerListRunning {
		runningContainers[c.ID] = c
		delete(nonrunningContainers, c.ID)
	}
	return runningContainers, nonrunningContainers, nil
}
