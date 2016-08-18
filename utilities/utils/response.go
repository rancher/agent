package utils

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/docker/engine-api/types"
	"github.com/rancher/agent/utilities/config"
	"github.com/rancher/agent/utilities/docker"
	revents "github.com/rancher/event-subscriber/events"
	"golang.org/x/net/context"
)

func GetResponseData(event *revents.Event) map[string]interface{} {
	resourceType := event.ResourceType
	switch resourceType {
	case "instanceHostMap":
		return map[string]interface{}{resourceType: getInstanceHostMapData(event)}
	case "volumeStoragePoolMap":
		return map[string]interface{}{
			resourceType: map[string]interface{}{
				"volume": map[string]interface{}{
					"format": "docker",
				},
			},
		}
	case "instancePull":
		return map[string]interface{}{
			"fields": map[string]interface{}{
				"dockerImage": getInstancePullData(event),
			},
		}
	case "imageStoragePoolMap":
		return map[string]interface{}{
			resourceType: map[string]interface{}{},
		}
	default:
		return map[string]interface{}{}
	}

}

func getInstanceHostMapData(event *revents.Event) map[string]interface{} {
	instance, _ := GetInstanceAndHost(event)
	client := docker.DefaultClient
	var inspect types.ContainerJSON
	container := GetContainer(client, instance, false)
	dockerPorts := []string{}
	dockerIP := ""
	dockerMounts := []types.MountPoint{}
	if container != nil {
		inspect, _ = client.ContainerInspect(context.Background(), container.ID)
		dockerMounts = getMountData(container.ID)
		dockerIP = inspect.NetworkSettings.IPAddress
		if container.Ports != nil && len(container.Ports) > 0 {
			for _, port := range container.Ports {
				privatePort := fmt.Sprintf("%v/%v", port.PrivatePort, port.Type)
				portSpec := privatePort
				bindAddr := ""
				if port.IP != "" {
					bindAddr = fmt.Sprintf("%s:", port.IP)
				}
				publicPort := ""
				if port.PublicPort > 0 {
					publicPort = fmt.Sprintf("%v:", port.PublicPort)
				} else if port.IP != "" {
					publicPort = ":"
				}
				portSpec = bindAddr + publicPort + portSpec
				dockerPorts = append(dockerPorts, portSpec)
			}
		}
	}
	update := map[string]interface{}{
		"instance": map[string]interface{}{
			"+data": map[string]interface{}{
				"dockerContainer": container,
				"dockerInspect":   inspect,
				"+fields": map[string]interface{}{
					"dockerHostIp": config.DockerHostIP(),
					"dockerPorts":  dockerPorts,
					"dockerIp":     dockerIP,
				},
			},
		},
	}
	if container != nil {
		in, _ := GetFieldsIfExist(update, "instance")
		instanceMap := InterfaceToMap(in)
		instanceMap["externalId"] = container.ID
	}
	if dockerMounts != nil {
		da, _ := GetFieldsIfExist(update, "instance", "+data")
		dataMap := InterfaceToMap(da)
		dataMap["dockerMounts"] = dockerMounts
	}
	return update
}

func getMountData(containerID string) []types.MountPoint {
	client := docker.DefaultClient
	inspect, err := client.ContainerInspect(context.Background(), containerID)
	if err != nil {
		logrus.Error(err)
		return []types.MountPoint{}
	}
	return inspect.Mounts
}

func getInstancePullData(event *revents.Event) types.ImageInspect {
	imageName, _ := GetFieldsIfExist(event.Data, "instancePull", "image", "data", "dockerImage", "fullName")
	tag, _ := GetFieldsIfExist(event.Data, "instancePull", "tag")
	inspect, _, _ := docker.DefaultClient.ImageInspectWithRaw(context.Background(),
		fmt.Sprintf("%v%v", imageName, tag), false)
	return inspect
}
