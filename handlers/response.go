package handlers

import (
	dclient "github.com/docker/docker/client"
	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	"github.com/rancher/agent/utils/constants"
	"github.com/rancher/agent/utils/utils"
	revents "github.com/rancher/event-subscriber/events"
	"github.com/rancher/go-rancher/v2"
)

func instanceHostMapReply(event *revents.Event, client *client.RancherClient, dockerClient *dclient.Client, cache *cache.Cache) error {
	data, err := utils.InstanceHostMapReply(event, dockerClient, cache)
	if err != nil {
		return errors.Wrap(err, constants.InstanceHostMapReplyError+"failed to get reply data")
	}
	return reply(data, event, client)
}

func instancePullReply(event *revents.Event, client *client.RancherClient, dockerClient *dclient.Client) error {
	data, err := utils.InstancePullReply(event, dockerClient)
	if err != nil {
		return errors.Wrap(err, constants.InstancePullReplyError+"failed to get reply data")
	}
	return reply(data, event, client)
}

func volumeStoragePoolMapReply(event *revents.Event, client *client.RancherClient) error {
	data, _ := utils.VolumeStoragePoolMapReply()
	return reply(data, event, client)
}
