package ping

import (
	"github.com/Sirupsen/logrus"
	"github.com/docker/engine-api/types"
	"github.com/nu7hatch/gouuid"
	"github.com/rancher/agent/core/hostInfo"
	"github.com/rancher/agent/utilities/config"
	"github.com/rancher/agent/utilities/docker"
	"github.com/rancher/agent/utilities/utils"
	revents "github.com/rancher/event-subscriber/events"
	"golang.org/x/net/context"
	"net"
)

func ReplyData(event *revents.Event) *revents.Event {
	var result revents.Event
	if event.ReplyTo != "" {
		value, _ := uuid.NewV4()
		result = revents.Event{
			ID:            value.String(),
			Name:          event.ReplyTo,
			Data:          map[string]interface{}{},
			ResourceType:  event.ResourceType,
			ResourceID:    event.ResourceID,
			PreviousIds:   event.ID,
			PreviousNames: event.Name,
		}
	}
	return &result
}

func DoPingAction(event, resp *revents.Event) {
	if !config.DockerEnable() {
		return
	}
	addResource(event, resp)
	addInstance(event, resp)
}

func addResource(ping, pong *revents.Event) {
	if !pingIncludeResource(ping) {
		return
	}
	stats := map[string]interface{}{}
	if pingIncludeStats(ping) {
		data := hostInfo.CollectData()
		stats = data
	}

	physicalHost := config.PhysicalHost()

	compute := map[string]interface{}{
		"type":             "host",
		"kind":             "docker",
		"hostname":         config.Hostname(),
		"createLabels":     config.Labels(),
		"labels":           getHostLabels(),
		"uuid":             config.DockerUUID(),
		"info":             stats,
		"physicalHostUuid": physicalHost["uuid"],
	}

	pool := map[string]interface{}{
		"type":     "storagePool",
		"kind":     "docker",
		"name":     utils.InterfaceToString(compute["hostname"]) + " Storage Pool",
		"hostUuid": utils.InterfaceToString(compute["uuid"]),
		"uuid":     utils.InterfaceToString(compute["uuid"]) + "-pool",
	}

	resolvedIP, err := net.LookupIP(config.DockerHostIP())
	if err != nil {
		logrus.Error(err)
	}

	ip := map[string]interface{}{
		"type":     "ipAddress",
		"uuid":     resolvedIP,
		"addresss": resolvedIP,
		"hostUuid": compute["uuid"],
	}
	proxy := config.HostProxy()
	if proxy != "" {
		compute["apiProxy"] = proxy
	}
	pingAddResource(pong, physicalHost, compute, pool, ip)
}

func addInstance(ping, pong *revents.Event) {
	if !pingIncludeInstance(ping) {
		return
	}
	if _, ok := utils.GetFieldsIfExist(pong.Data, "resources"); !ok {
		pong.Data["resources"] = []map[string]interface{}{}
	}
	pong.Data["resources"] = append(pong.Data["resources"].([]map[string]interface{}), map[string]interface{}{
		"type": "hostUuid",
		"uuid": config.DockerUUID(),
	})
	containers := []map[string]interface{}{}
	running, nonrunning := getAllContainerByState()
	for _, container := range running {
		containers = utils.AddContainer("running", &container, containers)
	}
	for _, container := range nonrunning {
		containers = utils.AddContainer("stopped", &container, containers)
	}
	if _, ok := utils.GetFieldsIfExist(pong.Data, "resources"); !ok {
		pong.Data["resources"] = []map[string]interface{}{}
	}
	for _, container := range containers {
		pong.Data["resources"] = append(pong.Data["resources"].([]map[string]interface{}), container)
	}
	pingSetOptions(pong, "instances", true)
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

func getHostLabels() map[string]string {
	return hostInfo.HostLabels("io.rancher.host")
}

func pingAddResource(pong *revents.Event, physcialHost map[string]interface{},
	compute map[string]interface{}, pool map[string]interface{}, ip map[string]interface{}) {
	if _, ok := utils.GetFieldsIfExist(pong.Data, "resources"); !ok {
		pong.Data["resources"] = []map[string]interface{}{}
	}
	pong.Data["resources"] = append(pong.Data["resources"].([]map[string]interface{}), physcialHost)
	pong.Data["resources"] = append(pong.Data["resources"].([]map[string]interface{}), compute)
	pong.Data["resources"] = append(pong.Data["resources"].([]map[string]interface{}), pool)
	pong.Data["resources"] = append(pong.Data["resources"].([]map[string]interface{}), ip)
	logrus.Infof("debug pong %v", physcialHost)
}

func getAllContainerByState() (map[string]types.Container, map[string]types.Container) {
	client := docker.DefaultClient
	nonrunningContainers := map[string]types.Container{}
	containerList, _ := client.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	for _, c := range containerList {
		if c.Status != "" && c.Status != "Created" {
			nonrunningContainers[c.ID] = c
		}
	}
	runningContainers := map[string]types.Container{}
	containerListRunning, _ := client.ContainerList(context.Background(), types.ContainerListOptions{})
	for _, c := range containerListRunning {
		runningContainers[c.ID] = c
		delete(nonrunningContainers, c.ID)
	}
	return runningContainers, nonrunningContainers
}

func pingSetOptions(pong *revents.Event, key string, value bool) {
	if _, ok := pong.Data["options"]; !ok {
		pong.Data["options"] = map[string]interface{}{}
	}
	pong.Data["options"].(map[string]interface{})[key] = value
}
