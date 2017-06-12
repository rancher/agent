package handlers

import (
	"github.com/Sirupsen/logrus"
	engineCli "github.com/docker/docker/client"
	"github.com/mitchellh/mapstructure"
	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	"github.com/rancher/agent/core/storage"
	"github.com/rancher/agent/model"
	"github.com/rancher/agent/utilities/constants"
	"github.com/rancher/agent/utilities/utils"
	revents "github.com/rancher/event-subscriber/events"
	"github.com/rancher/go-rancher/v2"
)

type StorageHandler struct {
	dockerClient *engineCli.Client
	cache        *cache.Cache
}

func (h *StorageHandler) VolumeActivate(event *revents.Event, cli *client.RancherClient) error {
	var volumeStoragePoolMap model.VolumeStoragePoolMap
	err := mapstructure.Decode(event.Data["volumeStoragePoolMap"], &volumeStoragePoolMap)
	if err != nil {
		return errors.Wrap(err, constants.VolumeActivateError+"failed to marshall incoming request")
	}
	volume := volumeStoragePoolMap.Volume
	storagePool := volumeStoragePoolMap.StoragePool
	progress := utils.GetProgress(event, cli)

	if ok, err := storage.IsVolumeActive(volume, storagePool, h.dockerClient); ok {
		return volumeStoragePoolMapReply(event, cli)
	} else if err != nil {
		return errors.Wrap(err, constants.VolumeActivateError+"failed to check whether volume is activated")
	}

	if err := storage.DoVolumeActivate(volume, storagePool, progress, h.dockerClient); err != nil {
		return errors.Wrap(err, constants.VolumeActivateError+"failed to activate volume")
	}
	if ok, err := storage.IsVolumeActive(volume, storagePool, h.dockerClient); !ok && err != nil {
		return errors.Wrap(err, constants.VolumeActivateError)
	} else if !ok && err == nil {
		return errors.New(constants.VolumeActivateError + "volume is not activated")
	}
	logrus.Infof("rancher id [%v]: Volume with name [%v] has been activated", event.ResourceID, volume.Name)
	return volumeStoragePoolMapReply(event, cli)
}

func (h *StorageHandler) VolumeRemove(event *revents.Event, cli *client.RancherClient) error {
	var volumeStoragePoolMap model.VolumeStoragePoolMap
	err := mapstructure.Decode(event.Data["volumeStoragePoolMap"], &volumeStoragePoolMap)
	if err != nil {
		return errors.Wrap(err, constants.VolumeRemoveError+"failed to marshall incoming request")
	}
	volume := volumeStoragePoolMap.Volume
	storagePool := volumeStoragePoolMap.StoragePool
	progress := utils.GetProgress(event, cli)

	if ok, err := storage.IsVolumeRemoved(volume, storagePool, h.dockerClient); err == nil && !ok {
		rmErr := storage.DoVolumeRemove(volume, storagePool, progress, h.dockerClient, h.cache, event.ResourceID)
		if rmErr != nil {
			return errors.Wrap(rmErr, constants.VolumeRemoveError+"failed to remove volume")
		}
	} else if err != nil {
		return errors.Wrap(err, constants.VolumeRemoveError+"failed to check whether volume is removed")
	}
	logrus.Infof("rancher id [%v]: Volume with name [%v] has been removed", event.ResourceID, volume.Name)
	return volumeStoragePoolMapReply(event, cli)
}
