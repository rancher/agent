package handlers

import (
	"github.com/Sirupsen/logrus"
	engineCli "github.com/docker/engine-api/client"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/rancher/agent/core/compute"
	"github.com/rancher/agent/core/storage"
	"github.com/rancher/agent/model"
	"github.com/rancher/agent/utilities/utils"
	revents "github.com/rancher/event-subscriber/events"
	"github.com/rancher/go-rancher/v2"
)

type StorageHandler struct {
	dockerClient *engineCli.Client
}

func (h *StorageHandler) ImageActivate(event *revents.Event, cli *client.RancherClient) error {
	var imageStoragePoolMap model.ImageStoragePoolMap
	if err := mapstructure.Decode(event.Data["imageStoragePoolMap"], &imageStoragePoolMap); err != nil {
		return errors.WithStack(err)
	}
	image := imageStoragePoolMap.Image
	storagePool := imageStoragePoolMap.StoragePool

	progress := utils.GetProgress(event, cli)

	if image.ID >= 0 && event.Data["processData"] != nil {
		image.ProcessData = event.Data["processData"].(map[string]interface{})
	}

	if ok, err := storage.IsImageActive(image, storagePool, h.dockerClient); ok {
		return h.reply(event, cli)
	} else if err != nil {
		return errors.WithStack(err)
	}

	err := storage.DoImageActivate(image, storagePool, progress, h.dockerClient, image.Name)
	if err != nil {
		return errors.WithStack(err)
	}

	if ok, err := storage.IsImageActive(image, storagePool, h.dockerClient); !ok && err != nil {
		return errors.WithStack(err)
	} else if !ok && err == nil {
		return errors.WithStack(err)
	}
	logrus.Infof("rancher id [%v]: Image with name [%v] has been activated", event.ResourceID, image.Name)
	return h.reply(event, cli)
}

func (h *StorageHandler) VolumeActivate(event *revents.Event, cli *client.RancherClient) error {
	var volumeStoragePoolMap model.VolumeStoragePoolMap
	err := mapstructure.Decode(event.Data["volumeStoragePoolMap"], &volumeStoragePoolMap)
	if err != nil {
		return errors.WithStack(err)
	}
	volume := volumeStoragePoolMap.Volume
	storagePool := volumeStoragePoolMap.StoragePool
	progress := utils.GetProgress(event, cli)

	if ok, err := storage.IsVolumeActive(volume, storagePool, h.dockerClient); ok {
		return h.reply(event, cli)
	} else if err != nil {
		return errors.WithStack(err)
	}

	if err := storage.DoVolumeActivate(volume, storagePool, progress, h.dockerClient); err != nil {
		return errors.WithStack(err)
	}
	if ok, err := storage.IsVolumeActive(volume, storagePool, h.dockerClient); !ok && err != nil {
		return errors.WithStack(err)
	} else if !ok && err == nil {
		return errors.WithStack(err)
	}
	logrus.Infof("rancher id [%v]: Volume with name [%v] has been activated", event.ResourceID, volume.Name)
	return h.reply(event, cli)
}

func (h *StorageHandler) VolumeDeactivate(event *revents.Event, cli *client.RancherClient) error {
	var volumeStoragePoolMap model.VolumeStoragePoolMap
	err := mapstructure.Decode(event.Data["volumeStoragePoolMap"], &volumeStoragePoolMap)
	if err != nil {
		return errors.WithStack(err)
	}
	volume := volumeStoragePoolMap.Volume
	storagePool := volumeStoragePoolMap.StoragePool
	progress := utils.GetProgress(event, cli)

	if storage.IsVolumeInactive(volume, storagePool) {
		return h.reply(event, cli)
	}

	if err := storage.DoVolumeDeactivate(volume, storagePool, progress); err != nil {
		return errors.WithStack(err)
	}
	if !storage.IsVolumeInactive(volume, storagePool) {
		return errors.WithStack(err)
	}
	return h.reply(event, cli)
}

func (h *StorageHandler) VolumeRemove(event *revents.Event, cli *client.RancherClient) error {
	var volumeStoragePoolMap model.VolumeStoragePoolMap
	err := mapstructure.Decode(event.Data["volumeStoragePoolMap"], &volumeStoragePoolMap)
	if err != nil {
		return errors.WithStack(err)
	}
	volume := volumeStoragePoolMap.Volume
	storagePool := volumeStoragePoolMap.StoragePool
	progress := utils.GetProgress(event, cli)

	if volume.DeviceNumber == 0 {
		if err := compute.PurgeState(volume.Instance, h.dockerClient); err != nil {
			return errors.WithStack(err)
		}
	}
	if ok, err := storage.IsVolumeRemoved(volume, storagePool, h.dockerClient); err == nil && !ok {
		rmErr := storage.DoVolumeRemove(volume, storagePool, progress, h.dockerClient)
		if rmErr != nil {
			return errors.WithStack(err)
		}
	} else if err != nil {
		return errors.WithStack(err)
	}
	logrus.Infof("rancher id [%v]: Volume with name [%v] has been removed", event.ResourceID, volume.Name)
	return h.reply(event, cli)
}

func (h *StorageHandler) reply(event *revents.Event, cli *client.RancherClient) error {
	resp, err := utils.GetResponseData(event, h.dockerClient)
	if err != nil {
		return errors.WithStack(err)
	}
	return reply(resp, event, cli)
}
