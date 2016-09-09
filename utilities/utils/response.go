package utils

import (
	"fmt"
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/pkg/errors"
	"github.com/rancher/agent/utilities/config"
	"github.com/rancher/agent/utilities/constants"
	revents "github.com/rancher/event-subscriber/events"
	"golang.org/x/net/context"
)

func GetResponseData(event *revents.Event, client *client.Client) (map[string]interface{}, error) {
	resourceType := event.ResourceType
	switch resourceType {
	case "instanceHostMap":
		resp, err := getInstanceHostMapData(event, client)
		if err != nil {
			return map[string]interface{}{}, errors.Wrap(err, constants.GetResponseDataError)
		}
		return map[string]interface{}{resourceType: resp}, nil
	case "volumeStoragePoolMap":
		return map[string]interface{}{
			resourceType: map[string]interface{}{
				"volume": map[string]interface{}{
					"format": "docker",
				},
			},
		}, nil
	case "instancePull":
		resp, err := getInstancePullData(event, client)
		if err != nil {
			return map[string]interface{}{}, errors.Wrap(err, constants.GetResponseDataError)
		}
		return map[string]interface{}{
			"fields": map[string]interface{}{
				"dockerImage": resp,
			},
		}, nil
	case "imageStoragePoolMap":
		return map[string]interface{}{
			resourceType: map[string]interface{}{},
		}, nil
	default:
		return map[string]interface{}{}, nil
	}

}

func getInstanceHostMapData(event *revents.Event, client *client.Client) (map[string]interface{}, error) {
	instance, _, err := GetInstanceAndHost(event)
	if err != nil {
		return map[string]interface{}{}, errors.Wrap(err, constants.GetInstanceHostMapDataError)
	}

	container, err := GetContainer(client, instance, false)
	if err != nil && !IsContainerNotFoundError(err) {
		return map[string]interface{}{}, errors.Wrap(err, constants.GetInstanceHostMapDataError)
	}

	if container.ID == "" {
		update := map[string]interface{}{
			"instance": map[string]interface{}{
				"+data": map[string]interface{}{
					"dockerInspect":   nil,
					"dockerContainer": nil,
					"+fields": map[string]interface{}{
						"dockerHostIp": config.DockerHostIP(),
						"dockerPorts":  nil,
						"dockerIp":     nil,
					},
				},
			},
		}
		in, _ := GetFieldsIfExist(update, "instance")
		instanceMap := InterfaceToMap(in)
		instanceMap["externalId"] = container.ID
		return update, nil
	}

	dockerPorts := []string{}
	dockerMounts := []types.MountPoint{}
	inspect, err := client.ContainerInspect(context.Background(), container.ID)
	if err != nil {
		return map[string]interface{}{}, errors.Wrap(err, constants.GetInstanceHostMapDataError)
	}
	dockerMounts, err = getMountData(container.ID, client)
	if err != nil {
		return map[string]interface{}{}, errors.Wrap(err, constants.GetInstanceHostMapDataError)
	}
	dockerIP := inspect.NetworkSettings.IPAddress
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

	in, _ := GetFieldsIfExist(update, "instance")
	instanceMap := InterfaceToMap(in)
	instanceMap["externalId"] = container.ID

	if dockerMounts != nil {
		da, _ := GetFieldsIfExist(update, "instance", "+data")
		dataMap := InterfaceToMap(da)
		dataMap["dockerMounts"] = dockerMounts
	}
	return update, nil
}

func getMountData(containerID string, client *client.Client) ([]types.MountPoint, error) {
	inspect, err := client.ContainerInspect(context.Background(), containerID)
	if err != nil {
		return []types.MountPoint{}, errors.Wrap(err, constants.GetMountDataError)
	}
	return inspect.Mounts, nil
}

func getInstancePullData(event *revents.Event, dockerClient *client.Client) (types.ImageInspect, error) {
	imageName, ok := GetFieldsIfExist(event.Data, "instancePull", "image", "data", "dockerImage", "fullName")
	if !ok {
		return types.ImageInspect{}, errors.New("Failed to get instance pull data: Can't get image name from event")
	}
	tag, ok := GetFieldsIfExist(event.Data, "instancePull", "tag")
	if !ok {
		return types.ImageInspect{}, errors.New("Failed to get instance pull data: Can't get image tag from event")
	}
	inspect, _, err := dockerClient.ImageInspectWithRaw(context.Background(),
		fmt.Sprintf("%v%v", imageName, tag))
	if err != nil && !client.IsErrImageNotFound(err) {
		return types.ImageInspect{}, errors.Wrap(err, constants.GetInstancePullDataError)
	}
	return inspect, nil
}
