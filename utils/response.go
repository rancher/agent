package utils

import (
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	revents "github.com/rancher/event-subscriber/events"
	"golang.org/x/net/context"
)

func InstancePullReply(event *revents.Event, client *client.Client) (map[string]interface{}, error) {
	resp, err := getInstancePullData(event, client)
	if err != nil {
		return map[string]interface{}{}, errors.Wrap(err, "failed to get instance pull data")
	}
	return map[string]interface{}{
		"fields": map[string]interface{}{
			"dockerImage": resp,
		},
	}, nil
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
		return types.ImageInspect{}, errors.Wrap(err, "failed to inspect images")
	}
	return inspect, nil
}
