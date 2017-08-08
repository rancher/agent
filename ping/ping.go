package ping

import (
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/rancher/agent/host_info"
	"github.com/rancher/agent/utils"
	revents "github.com/rancher/event-subscriber/events"
	"github.com/shirou/gopsutil/disk"
	"golang.org/x/net/context"
)

const (
	ipLabel           = "io.rancher.scheduler.ips"
	agentImage        = "RANCHER_AGENT_IMAGE"
	ipSet             = "CATTLE_SCHEDULER_IPS"
	requireAny        = "CATTLE_SCHEDULER_REQUIRE_ANY"
	requireAnyLabel   = "io.rancher.scheduler.require_any"
	RancherAgentImage = "io.rancher.host.agent_image"
)

type Response struct {
	Resources []Resource `json:"resources,omitempty" yaml:"resources,omitempty"`
	Options   Options    `json:"options,omitempty" yaml:"options,omitempty"`
}

type Options struct {
	Instances bool `json:"instances,omitempty" yaml:"instances,omitempty"`
}

type Resource struct {
	Type                           string                 `json:"type,omitempty" yaml:"type,omitempty"`
	Kind                           string                 `json:"kind,omitempty" yaml:"kind,omitempty"`
	HostName                       string                 `json:"hostname,omitempty" yaml:"hostname,omitempty"`
	CreateLabels                   map[string]string      `json:"createLabels,omitempty" yaml:"createLabels,omitempty"`
	Labels                         map[string]string      `json:"labels,omitempty" yaml:"labels,omitempty"`
	UUID                           string                 `json:"uuid,omitempty" yaml:"uuid,omitempty"`
	PhysicalHostUUID               string                 `json:"physicalHostUuid,omitempty" yaml:"physicalHostUuid,omitempty"`
	Info                           map[string]interface{} `json:"info,omitempty" yaml:"info,omitempty"`
	HostUUID                       string                 `json:"hostUuid,omitempty" yaml:"hostUuid,omitempty"`
	Name                           string                 `json:"name,omitempty" yaml:"name,omitempty"`
	Addresss                       string                 `json:"addresss,omitempty" yaml:"addresss,omitempty"`
	APIProxy                       string                 `json:"apiProxy,omitempty" yaml:"apiProxy,omitempty"`
	State                          string                 `json:"state,omitempty" yaml:"state,omitempty"`
	Image                          string                 `json:"image,omitempty" yaml:"image,omitempty"`
	Created                        int64                  `json:"created,omitempty" yaml:"created,omitempty"`
	Memory                         uint64                 `json:"memory,omitempty" yaml:"memory,omitempty"`
	MilliCPU                       uint64                 `json:"milliCpu,omitempty" yaml:"milli_cpu,omitempty"`
	LocalStorageMb                 uint64                 `json:"localStorageMb,omitempty" yaml:"local_storage_mb,omitempty"`
	ExternalID                     string                 `json:"externalId,omitempty" yaml:"externalId,omitempty"`
	MachineServiceRegistrationUUID string                 `json:"machineServiceRegistrationUuid,omitempty" yaml:"machineServiceRegistrationUuid,omitempty"`
}

func DoPingAction(event *revents.Event, resp *Response, dockerClient *client.Client, collectors []hostInfo.Collector) error {
	if !utils.DockerEnable() {
		return nil
	}
	if err := addResource(event, resp, collectors); err != nil {
		return errors.Wrap(err, "failed to add resource")
	}
	if err := addInstance(event, resp, dockerClient); err != nil {
		return errors.Wrap(err, "failed to add instance")
	}
	return nil
}

func addResource(ping *revents.Event, pong *Response, collectors []hostInfo.Collector) error {
	if !includeResource(ping) {
		return nil
	}
	stats := map[string]interface{}{}
	if includeStats(ping) {
		data := hostInfo.CollectData(collectors)
		stats = data
	}

	hostname, err := utils.Hostname()
	if err != nil {
		return errors.Wrap(err, "failed to get hostname")
	}

	physicalHost, err := physicalHost(hostname)
	if err != nil {
		return errors.Wrap(err, "failed to get physical host")
	}

	labels, err := getHostLabels(collectors)
	if err != nil {
		logrus.Warnf("Failed to get Host Labels err msg: %v", err.Error())
	}
	rancherImage := os.Getenv(agentImage)
	createLabels := utils.Labels()
	if os.Getenv(ipSet) != "" {
		createLabels[ipLabel] = os.Getenv(ipSet)
	}
	if os.Getenv(requireAny) != "" {
		createLabels[requireAnyLabel] = os.Getenv(requireAny)
	}
	labels[RancherAgentImage] = rancherImage
	compute := Resource{
		Type:                           "host",
		Kind:                           "docker",
		HostName:                       hostname,
		CreateLabels:                   createLabels,
		Labels:                         labels,
		Info:                           stats,
		APIProxy:                       utils.HostProxy(),
		MachineServiceRegistrationUUID: os.Getenv("CATTLE_PHYSICAL_HOST_UUID"),
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

	pool := Resource{
		Type: "storagePool",
		Kind: "docker",
		Name: compute.HostName + " Storage Pool",
	}

	resolvedIP, err := net.LookupIP(utils.DockerHostIP())
	ipAddr := ""
	if err != nil {
		logrus.Warn(err)
	} else {
		ipAddr = resolvedIP[0].String()
	}
	ip := Resource{
		Type:     "ipAddress",
		UUID:     ipAddr,
		Addresss: ipAddr,
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

func addInstance(ping *revents.Event, pong *Response, dockerClient *client.Client) error {
	if !includeInstance(ping) {
		return nil
	}
	containers := []Resource{}

	// if we can not get all container in 2s, we will skip it
	done := make(chan bool, 1)
	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(2 * time.Second)
		timeout <- true
	}()
	running, nonRunning, err := getAllContainerByState(dockerClient, done)
	select {
	case <-done:
		if err != nil {
			return errors.Wrap(err, "failed to get all containers")
		}
		for _, container := range running {
			containers = addContainer("running", container, containers)
		}
		for _, container := range nonRunning {
			containers = addContainer("stopped", container, containers)
		}
		pong.Resources = append(pong.Resources, containers...)
		pong.Options.Instances = true
		return nil
	case <-timeout:
		logrus.Warn("Can not get response from docker daemon")
		return nil
	}
}

func includeResource(ping *revents.Event) bool {
	value, ok := utils.GetFieldsIfExist(ping.Data, "options", "resources")
	if !ok {
		return false
	}
	return utils.InterfaceToBool(value)
}

func includeStats(ping *revents.Event) bool {
	value, ok := utils.GetFieldsIfExist(ping.Data, "options", "stats")
	if !ok {
		return false
	}
	return utils.InterfaceToBool(value)
}

func includeInstance(ping *revents.Event) bool {
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
	nonRunningContainers := map[string]types.Container{}
	containerList, err := dockerClient.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		return map[string]types.Container{}, map[string]types.Container{}, errors.Wrap(err, "failed to list containers")
	}
	for _, c := range containerList {
		if c.Status != "" && c.Status != "Created" {
			nonRunningContainers[c.ID] = c
		}
	}
	runningContainers := map[string]types.Container{}
	// if status is running, it is a running container
	for _, c := range containerList {
		if strings.Contains(c.Status, "Up") || c.State == "Running" {
			runningContainers[c.ID] = c
			delete(nonRunningContainers, c.ID)
		}
	}
	done <- true
	return runningContainers, nonRunningContainers, nil
}

func physicalHost(hostname string) (Resource, error) {
	return Resource{
		Type: "physicalHost",
		Kind: "physicalHost",
		Name: hostname,
	}, nil
}

func addContainer(state string, container types.Container, containers []Resource) []Resource {
	containerData := Resource{
		Type:       "instance",
		UUID:       container.Labels[utils.UUIDLabel],
		State:      state,
		ExternalID: container.ID,
	}
	return append(containers, containerData)
}
