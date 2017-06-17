package utils

import (
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	"github.com/rancher/agent/utils/constants"
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

func isRunning(id string, client *client.Client) (bool, error) {
	inspect, err := client.ContainerInspect(context.Background(), id)
	if err != nil {
		return false, err
	}
	return inspect.State.Pid != 0, nil
}

func getInstanceHostMapData(event *revents.Event, client *client.Client, cache *cache.Cache) (map[string]interface{}, error) {
	instance, err := GetInstance(event)
	if err != nil {
		return map[string]interface{}{}, errors.Wrap(err, constants.GetInstanceHostMapDataError+"failed to marshall instancehostmap")
	}

	container, err := GetContainer(client, instance, false)
	if err != nil && !IsContainerNotFoundError(err) {
		return map[string]interface{}{}, errors.Wrap(err, constants.GetInstanceHostMapDataError+"failed to get container")
	}

	if container.ID == "" {
		update := map[string]interface{}{
			"+data": map[string]interface{}{
				"dockerInspect": nil,
				"+fields": map[string]interface{}{
					"dockerIp": nil,
				},
			},
			"externalId": instance.ExternalID,
		}
		return update, nil
	}

	inspect, err := client.ContainerInspect(context.Background(), container.ID)
	if err != nil {
		return map[string]interface{}{}, errors.Wrap(err, constants.GetInstanceHostMapDataError+"failed to inspect container")
	}
	dockerIP, err := getIP(inspect, cache)
	if err != nil && !IsNoopEvent(event) {
		if running, err2 := isRunning(inspect.ID, client); err2 != nil {
			return nil, errors.Wrap(err2, constants.GetInstanceHostMapDataError+"failed to inspect running container")
		} else if running {
			return nil, errors.Wrap(err, constants.GetInstanceHostMapDataError+"failed to get ip of the container")
		}
	}

	update := map[string]interface{}{
		"+data": map[string]interface{}{
			"dockerInspect": inspect,
			"+fields": map[string]interface{}{
				"dockerIp": dockerIP,
			},
		},
		"externalId": container.ID,
	}

	return update, nil
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
