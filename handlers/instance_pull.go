package handlers

import (
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/rancher/agent/runtime"
	"github.com/rancher/agent/utils"
	revents "github.com/rancher/event-subscriber/events"
	v3 "github.com/rancher/go-rancher/v3"
	"golang.org/x/net/context"
)

func (h *ComputeHandler) InstancePull(event *revents.Event, cli *v3.RancherClient) error {
	progress := utils.GetProgress(event, cli)
	var instancePull runtime.InstancePull
	err := mapstructure.Decode(event.Data["instancePull"], &instancePull)
	if err != nil {
		return errors.Wrap(err, "failed to marshall incoming request")
	}
	imageUUID := instancePull.Image.Data.DockerImage.FullName
	if instancePull.Image.Data.DockerImage.Server != "" {
		imageUUID = instancePull.Image.Data.DockerImage.Server + "/" + imageUUID
	}
	imageParams := runtime.PullParams{
		Mode:      instancePull.Mode,
		Complete:  instancePull.Complete,
		Tag:       instancePull.Tag,
		ImageUUID: imageUUID,
	}

	cred := instancePull.Image.RegistryCredential
	_, pullErr := runtime.DoInstancePull(imageParams, progress, h.dockerClient, v3.Credential{
		PublicValue: cred.PublicValue,
		SecretValue: cred.SecretValue,
	})
	if pullErr != nil {
		return errors.Wrap(pullErr, "failed to pull instance")
	}
	return instancePullReply(event, cli, h.dockerClient)
}

func instancePullReply(event *revents.Event, client *v3.RancherClient, dockerClient *client.Client) error {
	data, err := pullReply(event, dockerClient)
	if err != nil {
		return errors.Wrap(err, "failed to get reply data")
	}
	return reply(data, event, client)
}

func pullReply(event *revents.Event, client *client.Client) (map[string]interface{}, error) {
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
	imageName, ok := utils.GetFieldsIfExist(event.Data, "instancePull", "image", "data", "dockerImage", "fullName")
	if !ok {
		return types.ImageInspect{}, errors.New("Failed to get instance pull data: Can't get image name from event")
	}
	serverName, ok := utils.GetFieldsIfExist(event.Data, "instancePull", "image", "data", "dockerImage", "server")
	if !ok {
		return types.ImageInspect{}, errors.New("Failed to get instance pull data: Can't get server name from event")
	}
	tag, ok := utils.GetFieldsIfExist(event.Data, "instancePull", "tag")
	if !ok {
		return types.ImageInspect{}, errors.New("Failed to get instance pull data: Can't get image tag from event")
	}
	inspect, _, err := dockerClient.ImageInspectWithRaw(context.Background(),
		fmt.Sprintf("%v/%v%v", serverName, imageName, tag))
	if err != nil && !client.IsErrImageNotFound(err) {
		return types.ImageInspect{}, errors.Wrap(err, "failed to inspect images")
	}
	return inspect, nil
}
