package handlers

import (
	"github.com/Sirupsen/logrus"
	engineCli "github.com/docker/engine-api/client"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/rancher/agent/core/compute"
	"github.com/rancher/agent/core/storage"
	"github.com/rancher/agent/model"
	"github.com/rancher/agent/utilities/constants"
	"github.com/rancher/agent/utilities/utils"
	revents "github.com/rancher/event-subscriber/events"
	"github.com/rancher/go-rancher/client"
)

type StorageHandler struct {
	dockerClient *engineCli.Client
}

func (h *StorageHandler) ImageActivate(event *revents.Event, cli *client.RancherClient) error {
	var imageStoragePoolMap model.ImageStoragePoolMap
	if err := mapstructure.Decode(event.Data["imageStoragePoolMap"], &imageStoragePoolMap); err != nil {
		return errors.Wrap(err, constants.ImageActivateError+"failed to marshall incoming request")
	}
	image := imageStoragePoolMap.Image
	storagePool := imageStoragePoolMap.StoragePool

	progress := utils.GetProgress(event, cli)

	if image.ID >= 0 && event.Data["processData"] != nil {
		image.ProcessData = event.Data["processData"].(map[string]interface{})
	}

	if ok, err := storage.IsImageActive(image, storagePool, h.dockerClient); ok {
		return h.reply(event, cli, constants.ImageActivateError)
	} else if err != nil {
		return errors.Wrap(err, constants.ImageActivateError+"failed to check whether image is activated")
	}

	err := storage.DoImageActivate(image, storagePool, progress, h.dockerClient)
	if err != nil {
		return errors.Wrap(err, constants.ImageActivateError+"failed to do image activate")
	}

	if ok, err := storage.IsImageActive(image, storagePool, h.dockerClient); !ok && err != nil {
		return errors.Wrap(err, constants.ImageActivateError+"failed to check whether image is activated")
	} else if !ok && err == nil {
		return errors.New(constants.ImageActivateError + "image is not activated")
	}
	logrus.Infof("rancher id [%v]: Image with name [%v] has been activated", event.ResourceID, image.Name)
	return h.reply(event, cli, constants.ImageActivateError)
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
		return h.reply(event, cli, constants.VolumeActivateError)
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
	return h.reply(event, cli, constants.VolumeActivateError)
}

func (h *StorageHandler) VolumeDeactivate(event *revents.Event, cli *client.RancherClient) error {
	var volumeStoragePoolMap model.VolumeStoragePoolMap
	err := mapstructure.Decode(event.Data["volumeStoragePoolMap"], &volumeStoragePoolMap)
	if err != nil {
		return errors.Wrap(err, constants.VolumeDeactivateError+"failed to marshall incoming request")
	}
	volume := volumeStoragePoolMap.Volume
	storagePool := volumeStoragePoolMap.StoragePool
	progress := utils.GetProgress(event, cli)

	if storage.IsVolumeInactive(volume, storagePool) {
		return h.reply(event, cli, constants.VolumeDeactivateError)
	}

	if err := storage.DoVolumeDeactivate(volume, storagePool, progress); err != nil {
		return errors.Wrap(err, constants.VolumeDeactivateError+"failed to deactivate volume")
	}
	if !storage.IsVolumeInactive(volume, storagePool) {
		return errors.New(constants.VolumeDeactivateError)
	}
	return h.reply(event, cli, constants.VolumeDeactivateError)
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

	if volume.DeviceNumber == 0 {
		if err := compute.PurgeState(volume.Instance, h.dockerClient); err != nil {
			return errors.Wrap(err, constants.VolumeRemoveError+"failed to purge state")
		}
	}
	if ok, err := storage.IsVolumeRemoved(volume, storagePool, h.dockerClient); err == nil && !ok {
		rmErr := storage.DoVolumeRemove(volume, storagePool, progress, h.dockerClient)
		if rmErr != nil {
			return errors.Wrap(rmErr, constants.VolumeRemoveError+"failed to remove volume")
		}
	} else if err != nil {
		return errors.Wrap(err, constants.VolumeRemoveError+"failed to check whether volume is removed")
	}
	logrus.Infof("rancher id [%v]: Volume with name [%v] has been removed", event.ResourceID, volume.Name)
	return h.reply(event, cli, constants.VolumeRemoveError)
}

func (h *StorageHandler) reply(event *revents.Event, cli *client.RancherClient, errMSG string) error {
	resp, err := utils.GetResponseData(event, h.dockerClient)
	if err != nil {
		return errors.Wrap(err, errMSG)
	}
	return reply(resp, event, cli)
}
