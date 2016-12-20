package ping

import (
	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/rancher/agent/core/hostInfo"
	"github.com/rancher/agent/model"
	"github.com/rancher/agent/utilities/config"
	"github.com/rancher/agent/utilities/constants"
	"github.com/rancher/agent/utilities/utils"
	revents "github.com/rancher/event-subscriber/events"
	"github.com/shirou/gopsutil/disk"
	"golang.org/x/net/context"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
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
		return errors.Wrap(err, constants.AddResourceError+"failed to get physical host")
	}

	hostname, err := config.Hostname()
	if err != nil {
		return errors.Wrap(err, constants.AddResourceError+"failed to get hostname")
	}
	labels, err := getHostLabels(collectors)
	if err != nil {
		logrus.Warnf("Failed to get Host Labels err msg: %v", err.Error())
	}
	rancherImage := os.Getenv("RANCHER_AGENT_IMAGE")
	labels[constants.RancherAgentImage] = rancherImage
	uuid, err := config.DockerUUID()
	if err != nil {
		return errors.Wrap(err, constants.AddResourceError+"failed to get docker UUID")
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

	if memOverride := getResourceOverride("CATTLE_MEMORY_OVERRIDE"); memOverride != 0 {
		compute.Memory = memOverride
	} else if stats["memoryInfo"] != nil {
		if memTotal, ok := stats["memoryInfo"].(map[string]interface{})["memTotal"]; ok {
			compute.Memory = memTotal.(uint64) * 1024 * 1024
		}
	}

	if cpuOverride := getResourceOverride("CATTLE_MILLI_CPU_OVERRIDE"); cpuOverride != 0 {
		compute.MilliCPU = cpuOverride
	} else if stats["cpuInfo"] != nil {
		if cpuCount, ok := stats["cpuInfo"].(map[string]interface{})["count"].(int); ok {
			compute.MilliCPU = uint64(cpuCount) * 1000
		}
	}

	if storageOverride := getResourceOverride("CATTLE_LOCAL_STORAGE_MB_OVERRIDE"); storageOverride != 0 {
		compute.LocalStorageMb = storageOverride
	} else {
		usage, err := disk.Usage(".")
		if err != nil {
			logrus.Errorf("Error getting local storage usage: %v", err)
		} else {
			compute.LocalStorageMb = uint64(usage.Free) / 1000
		}
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

func getResourceOverride(envVar string) uint64 {
	var resource uint64
	var err error
	if val := os.Getenv(envVar); val != "" {
		resource, err = strconv.ParseUint(val, 10, 64)
		if err != nil {
			logrus.Warnf("Couldn't parse %v %v. Will not use it.", envVar, val)
			return 0
		}
	}
	return resource
}

func addInstance(ping *revents.Event, pong *model.PingResponse, dockerClient *client.Client) error {
	if !pingIncludeInstance(ping) {
		return nil
	}
	uuid, err := config.DockerUUID()
	if err != nil {
		return errors.Wrap(err, constants.AddInstanceError+"failed to get docker UUID")
	}
	pong.Resources = append(pong.Resources, model.PingResource{
		Type: "hostUuid",
		UUID: uuid,
	})
	containers := []model.PingResource{}

	// if we can not get all container in 2s, we will skip it
	done := make(chan bool, 1)
	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(2 * time.Second)
		timeout <- true
	}()
	running, nonrunning, err := getAllContainerByState(dockerClient, done)
	select {
	case <-done:
		if err != nil {
			return errors.Wrap(err, constants.AddInstanceError+"failed to get all containers")
		}
		for _, container := range running {
			containers = utils.AddContainer("running", container, containers, dockerClient)
		}
		for _, container := range nonrunning {
			containers = utils.AddContainer("stopped", container, containers, dockerClient)
		}
		pong.Resources = append(pong.Resources, containers...)
		pong.Options.Instances = true
		return nil
	case <-timeout:
		logrus.Warn("Can not get response from docker daemon")
		return nil
	}
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

func getAllContainerByState(dockerClient *client.Client, done chan bool) (map[string]types.Container, map[string]types.Container, error) {
	// avoid calling API twice
	nonrunningContainers := map[string]types.Container{}
	containerList, err := dockerClient.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		return map[string]types.Container{}, map[string]types.Container{}, errors.Wrap(err, constants.GetAllContainerByStateError+"failed to list containers")
	}
	for _, c := range containerList {
		if c.Status != "" && c.Status != "Created" {
			nonrunningContainers[c.ID] = c
		}
	}
	runningContainers := map[string]types.Container{}
	// if status is running, it is a running container
	for _, c := range containerList {
		if strings.Contains(c.Status, "Up") || c.State == "Running" {
			runningContainers[c.ID] = c
			delete(nonrunningContainers, c.ID)
		}
	}
	done <- true
	return runningContainers, nonrunningContainers, nil
}
