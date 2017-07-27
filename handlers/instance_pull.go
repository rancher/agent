package handlers

import (
	"github.com/docker/docker/client"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/rancher/agent/runtime"
	"github.com/rancher/agent/utils"
	revents "github.com/rancher/event-subscriber/events"
	v2 "github.com/rancher/go-rancher/v2"
)

func (h *ComputeHandler) InstancePull(event *revents.Event, cli *v2.RancherClient) error {
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
	_, pullErr := runtime.DoInstancePull(imageParams, progress, h.dockerClient, &v2.DockerBuild{}, v2.Credential{
		PublicValue: cred.PublicValue,
		SecretValue: cred.SecretValue,
	})
	if pullErr != nil {
		return errors.Wrap(pullErr, "failed to pull instance")
	}
	return instancePullReply(event, cli, h.dockerClient)
}

func instancePullReply(event *revents.Event, client *v2.RancherClient, dockerClient *client.Client) error {
	data, err := utils.InstancePullReply(event, dockerClient)
	if err != nil {
		return errors.Wrap(err, "failed to get reply data")
	}
	return reply(data, event, client)
}
