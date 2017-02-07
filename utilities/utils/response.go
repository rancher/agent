package utils

import (
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	"github.com/rancher/agent/utilities/config"
	"github.com/rancher/agent/utilities/constants"
	revents "github.com/rancher/event-subscriber/events"
	"golang.org/x/net/context"
)

func InstanceHostMapReply(event *revents.Event, client *client.Client, cache *cache.Cache) (map[string]interface{}, error) {
	resp, err := getInstanceHostMapData(event, client, cache)
	if err != nil {
		return map[string]interface{}{}, errors.Wrap(err, constants.GetResponseDataError+"failed to get instance host map")
	}
	return map[string]interface{}{event.ResourceType: resp}, nil
}

func VolumeStoragePoolMapReply() (map[string]interface{}, error) {
	return map[string]interface{}{
		"volumeStoragePoolMap": map[string]interface{}{
			"volume": map[string]interface{}{
				"format": "docker",
			},
		},
	}, nil
}

func InstancePullReply(event *revents.Event, client *client.Client) (map[string]interface{}, error) {
	resp, err := getInstancePullData(event, client)
	if err != nil {
		return map[string]interface{}{}, errors.Wrap(err, constants.GetResponseDataError+"failed to get instance pull data")
	}
	return map[string]interface{}{
		"fields": map[string]interface{}{
			"dockerImage": resp,
		},
	}, nil
}

func ImageStoragePoolMapReply() (map[string]interface{}, error) {
	return map[string]interface{}{
		"imageStoragePoolMap": map[string]interface{}{},
	}, nil
}

func isRunning(id string, client *client.Client) (bool, error) {
	inspect, err := client.ContainerInspect(context.Background(), id)
	if err != nil {
		return false, err
	}
	return inspect.State.Pid != 0, nil
}

func getInstanceHostMapData(event *revents.Event, client *client.Client, cache *cache.Cache) (map[string]interface{}, error) {
	instance, _, err := GetInstanceAndHost(event)
	if err != nil {
		return map[string]interface{}{}, errors.Wrap(err, constants.GetInstanceHostMapDataError+"failed to marshall instancehostmap")
	}

	container, err := GetContainer(client, instance, false)
	if err != nil && !IsContainerNotFoundError(err) {
		return map[string]interface{}{}, errors.Wrap(err, constants.GetInstanceHostMapDataError+"failed to get container")
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
		// in this case agent can't find the container so externalID will be populated from instance data
		instanceMap["externalId"] = instance.ExternalID
		return update, nil
	}

	dockerPorts := []string{}
	dockerMounts := []types.MountPoint{}
	inspect, err := client.ContainerInspect(context.Background(), container.ID)
	if err != nil {
		return map[string]interface{}{}, errors.Wrap(err, constants.GetInstanceHostMapDataError+"failed to inspect container")
	}
	dockerMounts, err = getMountData(container.ID, client)
	if err != nil {
		return map[string]interface{}{}, errors.Wrap(err, constants.GetInstanceHostMapDataError+"failed to get mount data")
	}
	if err := setupDNS(inspect.ID); err != nil {
		return nil, errors.Wrap(err, "Failed to set DNS client server addresses")
	}
	dockerIP, err := getIP(inspect, cache)
	if err != nil && !IsNoopEvent(event) {
		if running, err2 := isRunning(inspect.ID, client); err2 != nil {
			return nil, errors.Wrap(err2, constants.GetInstanceHostMapDataError+"failed to inspect running container")
		} else if running {
			return nil, errors.Wrap(err, constants.GetInstanceHostMapDataError+"failed to get ip of the container")
		}
	}
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

	if len(dockerPorts) == 0 {
		image, _, err := client.ImageInspectWithRaw(context.Background(), container.ImageID)
		if err == nil && image.Config != nil {
			for k := range image.Config.ExposedPorts {
				dockerPorts = append(dockerPorts, string(k))
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
		return []types.MountPoint{}, errors.Wrap(err, constants.GetMountDataError+"failed to inspect container")
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
		return types.ImageInspect{}, errors.Wrap(err, constants.GetInstancePullDataError+"failed to inspect images")
	}
	return inspect, nil
}
