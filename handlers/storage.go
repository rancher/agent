package handlers

import (
	"github.com/Sirupsen/logrus"
	engineCli "github.com/docker/docker/client"
	"github.com/mitchellh/mapstructure"
	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	"github.com/rancher/agent/core/storage"
	"github.com/rancher/agent/model"
	"github.com/rancher/agent/utils/constants"
	revents "github.com/rancher/event-subscriber/events"
	"github.com/rancher/go-rancher/v2"
)

type StorageHandler struct {
	dockerClient *engineCli.Client
	cache        *cache.Cache
}

func (h *StorageHandler) VolumeRemove(event *revents.Event, cli *client.RancherClient) error {
	volume := model.Volume{}
	err := mapstructure.Decode(event.Data["volume"], &volume)
	if err != nil {
		return errors.Wrap(err, constants.VolumeRemoveError+"failed to marshall incoming request")
	}

	if ok, err := storage.IsVolumeRemoved(volume, h.dockerClient); err == nil && !ok {
		rmErr := storage.DoVolumeRemove(volume, h.dockerClient, h.cache, event.ResourceID)
		if rmErr != nil {
			return errors.Wrap(rmErr, constants.VolumeRemoveError+"failed to remove volume")
		}
	} else if err != nil {
		return errors.Wrap(err, constants.VolumeRemoveError+"failed to check whether volume is removed")
	}
	logrus.Infof("rancher id [%v]: Volume with name [%v] has been removed", event.ResourceID, volume.Name)
	return volumeStoragePoolMapReply(event, cli)
}
