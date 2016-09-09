package ping

import (
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
	"net"
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
		return errors.Wrap(err, constants.AddResourceError)
	}

	hostname, err := config.Hostname()
	if err != nil {
		return errors.Wrap(err, constants.AddResourceError)
	}
	labels, err := getHostLabels(collectors)
	if err != nil {
		logrus.Warnf("Failed to get Host Labels err msg: %v", err.Error())
	}
	uuid, err := config.DockerUUID()
	if err != nil {
		return errors.Wrap(err, constants.AddResourceError)
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

func addInstance(ping *revents.Event, pong *model.PingResponse, dockerClient *client.Client) error {
	if !pingIncludeInstance(ping) {
		return nil
	}
	uuid, err := config.DockerUUID()
	if err != nil {
		return errors.Wrap(err, constants.AddInstanceError)
	}
	pong.Resources = append(pong.Resources, model.PingResource{
		Type: "hostUuid",
		UUID: uuid,
	})
	containers := []model.PingResource{}
	running, nonrunning, err := getAllContainerByState(dockerClient)
	if err != nil {
		return errors.Wrap(err, constants.AddInstanceError)
	}
	for _, container := range running {
		containers, err = utils.AddContainer("running", container, containers, dockerClient)
		if err != nil {
			return errors.Wrap(err, constants.AddInstanceError)
		}
	}
	for _, container := range nonrunning {
		containers, err = utils.AddContainer("stopped", container, containers, dockerClient)
		if err != nil {
			return errors.Wrap(err, constants.AddInstanceError)
		}
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
		return map[string]types.Container{}, map[string]types.Container{}, errors.Wrap(err, constants.GetAllContainerByStateError)
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
