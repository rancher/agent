package utils

import (
	"github.com/Sirupsen/logrus"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/filters"
	"github.com/nu7hatch/gouuid"
	"github.com/rancher/agent/handlers/docker"
	"github.com/rancher/agent/handlers/hostInfo"
	revents "github.com/rancher/go-machine-service/events"
	"golang.org/x/net/context"
	"net"
	"strings"
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
	if !DockerEnable() {
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

	physicalHost := physicalHost()

	compute := map[string]interface{}{
		"type":         "host",
		"kind":         "docker",
		"hostname":     hostname(),
		"createLabels": labels(),
		"labels":       getHostLabels(),
		"uuid":         dockerUUID(),
		"info":         stats,
	}

	pool := map[string]interface{}{
		"type":     "storagePool",
		"kind":     "docker",
		"name":     compute["hostname"].(string) + " Storage Pool",
		"hostUuid": compute["uuid"].(string),
		"uuid":     compute["uuid"].(string) + "-pool",
	}

	resolvedIP, err := net.LookupIP(DockerHostIP())
	if err != nil {
		logrus.Error(err)
	}

	ip := map[string]interface{}{
		"type":     "ipAddress",
		"uuid":     resolvedIP,
		"addresss": resolvedIP,
		"hostUuid": compute["uuid"],
	}
	proxy := hostProxy()
	if proxy != "" {
		compute["apiPorxy"] = proxy
	}
	pingAddResource(pong, physicalHost, compute, pool, ip)
}

func addInstance(ping, pong *revents.Event) {
	if !pingIncludeInstance(ping) {
		return
	}
	if _, ok := GetFieldsIfExist(pong.Data, "resources"); !ok {
		pong.Data["resource"] = []map[string]interface{}{}
	}
	pong.Data["resources"] = append(pong.Data["resources"].([]map[string]interface{}), map[string]interface{}{
		"type": "hostUuid",
		"uuid": dockerUUID(),
	})
	containers := []map[string]interface{}{}
	running, nonrunning := getAllContainerByState()
	logrus.Infof("running containers %v", running)
	logrus.Infof("nonruning containers %v", nonrunning)
	for _, container := range running {
		containers = addContainer("running", &container, containers)
	}
	for _, container := range nonrunning {
		containers = addContainer("stopped", &container, containers)
	}
	if _, ok := GetFieldsIfExist(pong.Data, "resources"); !ok {
		pong.Data["resources"] = []map[string]interface{}{}
	}
	logrus.Infof("containers %v", containers)
	for _, container := range containers {
		pong.Data["resources"] = append(pong.Data["resources"].([]map[string]interface{}), container)
	}
	pingSetOptions(pong, "instances", true)
}

func pingIncludeResource(ping *revents.Event) bool {
	value, ok := GetFieldsIfExist(ping.Data, "options", "resources")
	if !ok {
		return false
	}
	return value.(bool)
}

func pingIncludeStats(ping *revents.Event) bool {
	value, ok := GetFieldsIfExist(ping.Data, "options", "stats")
	if !ok {
		return false
	}
	return value.(bool)
}

func pingIncludeInstance(ping *revents.Event) bool {
	value, ok := GetFieldsIfExist(ping.Data, "options", "instances")
	if !ok {
		return false
	}
	return value.(bool)
}

func getHostLabels() map[string]string {
	return hostInfo.HostLabels("io.rancher.host")
}

func pingAddResource(pong *revents.Event, physcialHost map[string]interface{},
	compute map[string]interface{}, pool map[string]interface{}, ip map[string]interface{}) {
	if _, ok := GetFieldsIfExist(pong.Data, "resources"); !ok {
		pong.Data["resources"] = []map[string]interface{}{}
	}
	pong.Data["resources"] = append(pong.Data["resources"].([]map[string]interface{}), physcialHost)
	pong.Data["resources"] = append(pong.Data["resources"].([]map[string]interface{}), compute)
	pong.Data["resources"] = append(pong.Data["resources"].([]map[string]interface{}), pool)
	pong.Data["resources"] = append(pong.Data["resources"].([]map[string]interface{}), ip)
}

func getAllContainerByState() (map[string]types.Container, map[string]types.Container) {
	client := docker.GetClient(DefaultVersion)
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

func addContainer(state string, container *types.Container, containers []map[string]interface{}) []map[string]interface{} {
	labels := container.Labels

	containerData := map[string]interface{}{
		"type":            "instance",
		"uuid":            getUUID(container),
		"state":           state,
		"systemContainer": getSysContainer(container),
		"dockerId":        container.ID,
		"image":           container.Image,
		"labels":          labels,
		"created":         container.Created,
	}
	return append(containers, containerData)
}

func getSysContainer(container *types.Container) string {
	image := container.Image
	systemImages := getAgentImage()
	if hasKey(systemImages, image) {
		return systemImages[image].(string)
	}
	label, ok := container.Labels["io.rancher.container.system"]
	if ok {
		return label
	}
	return ""
}

func getAgentImage() map[string]interface{} {
	client := docker.GetClient(DefaultVersion)
	args := filters.NewArgs()
	args.Add("label", SystemLables)
	images, _ := client.ImageList(context.Background(), types.ImageListOptions{Filters: args})
	systemImage := map[string]interface{}{}
	for _, image := range images {
		labelValue := image.Labels[SystemLables]
		for _, l := range image.RepoTags {
			if strings.HasSuffix(l, ":latest") {
				alias := l[:len(l)-7]
				systemImage[alias] = labelValue
			}
		}
	}
	return systemImage
}

func pingSetOptions(pong *revents.Event, key string, value bool) {
	if _, ok := pong.Data["options"]; !ok {
		pong.Data["options"] = map[string]interface{}{}
	}
	pong.Data["options"].(map[string]interface{})[key] = value
}
