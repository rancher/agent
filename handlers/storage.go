package handlers

import (
	"context"

	engineCli "github.com/docker/docker/client"
	"github.com/leodotcloud/log"
	"github.com/mitchellh/mapstructure"
	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	"github.com/rancher/agent/core/image"
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

func (h *StorageHandler) ImageActivate(event *revents.Event, cli *client.RancherClient) error {
	var imageStoragePoolMap model.ImageStoragePoolMap
	if err := mapstructure.Decode(event.Data["imageStoragePoolMap"], &imageStoragePoolMap); err != nil {
		return errors.Wrap(err, constants.ImageActivateError+"failed to marshall incoming request")
	}
	im := imageStoragePoolMap.Image
	storagePool := imageStoragePoolMap.StoragePool

	progress := utils.GetProgress(event, cli)

	if im.ID >= 0 && event.Data["processData"] != nil {
		if err := mapstructure.Decode(event.Data["processData"], &im.ProcessData); err != nil {
			return errors.Wrap(err, constants.ImageActivateError+"failed to marshall image process data")
		}
	}

	if ok, err := image.IsImageActive(im, storagePool, h.dockerClient); ok {
		return imageStoragePoolMapReply(event, cli)
	} else if err != nil {
		return errors.Wrap(err, constants.ImageActivateError+"failed to check whether image is activated")
	}

	err := image.DoImageActivate(im, storagePool, progress, h.dockerClient, im.Name)
	if err != nil {
		return errors.Wrap(err, constants.ImageActivateError+"failed to do image activate")
	}

	if ok, err := image.IsImageActive(im, storagePool, h.dockerClient); !ok && err != nil {
		return errors.Wrap(err, constants.ImageActivateError+"failed to check whether image is activated")
	} else if !ok && err == nil {
		return errors.New(constants.ImageActivateError + "failed to activate image")
	}
	log.Infof("rancher id [%v]: Image with name [%v] has been activated", event.ResourceID, im.Name)
	return imageStoragePoolMapReply(event, cli)
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

	// if its rancher volume, use flexVolume and bypass docker volume plugin
	if ok, err := storage.IsRancherVolume(volume); err != nil {
		return err
	} else if ok {
		err := storage.VolumeActivateFlex(volume)
		if err != nil {
			return err
		}
	} else {
		err := storage.VolumeActivateDocker(volume, storagePool, progress, h.dockerClient)
		if err != nil {
			return err
		}
	}
	log.Infof("rancher id [%v]: Volume with name [%v] has been activated", event.ResourceID, volume.Name)
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

	if ok, err := storage.IsRancherVolume(volume); err != nil {
		return err
	} else if ok {
		// we need to make sure the reference is cleaned up in docker, so if it exists in docker then clean up in docker first
		if inspect, err := h.dockerClient.VolumeInspect(context.Background(), volume.Name); err != nil && !engineCli.IsErrVolumeNotFound(err) {
			return errors.Wrapf(err, constants.VolumeRemoveError+"failed to inspect volume %v", volume.Name)
		} else if err == nil {
			err := h.dockerClient.VolumeRemove(context.Background(), inspect.Name, false)
			if err != nil {
				return errors.Wrapf(err, constants.VolumeRemoveError+"failed to remove volume %v", volume.Name)
			}
		} else {
			err := storage.VolumeRemoveFlex(volume)
			if err != nil {
				return errors.Wrapf(err, constants.VolumeRemoveError+"failed to remove volume %v in flex mode", volume.Name)
			}
		}
	} else {
		err := storage.VolumeRemoveDocker(volume, storagePool, progress, h.dockerClient, h.cache, event.ResourceID)
		if err != nil {
			return err
		}
	}
	log.Infof("rancher id [%v]: Volume with name [%v] has been removed", event.ResourceID, volume.Name)
	return volumeStoragePoolMapReply(event, cli)
}
