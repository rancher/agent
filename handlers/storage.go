package handlers

import (
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/rancher/agent/core/compute"
	"github.com/rancher/agent/core/progress"
	"github.com/rancher/agent/core/storage"
	"github.com/rancher/agent/model"
	"github.com/rancher/agent/utilities/utils"
	revents "github.com/rancher/event-subscriber/events"
	"github.com/rancher/go-rancher/client"
)

// TODO: similar to compute turn into struct w/ dockerclient as field
func ImageActivate(event *revents.Event, cli *client.RancherClient) error {
	var imageStoragePoolMap model.ImageStoragePoolMap
	mapstructure.Decode(event.Data["imageStoragePoolMap"], &imageStoragePoolMap)
	image := imageStoragePoolMap.Image
	storagePool := imageStoragePoolMap.StoragePool

	progress := progress.Progress{Request: event, Client: cli}

	if image.ID >= 0 && event.Data["processData"] != nil {
		image.ProcessData = event.Data["processData"].(map[string]interface{})
	}
	if storage.IsImageActive(&image, &storagePool) {
		return reply(utils.GetResponseData(event), event, cli)
	}

	err := storage.DoImageActivate(&image, &storagePool, &progress)
	if err != nil {
		return errors.Wrap(err, "Failed to activate image")
	}

	if !storage.IsImageActive(&image, &storagePool) {
		return errors.New("operation failed")
	}

	return reply(utils.GetResponseData(event), event, cli)
}

func VolumeActivate(event *revents.Event, cli *client.RancherClient) error {
	var volumeStoragePoolMap model.VolumeStoragePoolMap
	mapstructure.Decode(event.Data["volumeStoragePoolMap"], &volumeStoragePoolMap)
	volume := volumeStoragePoolMap.Volume
	storagePool := volumeStoragePoolMap.StoragePool
	progress := progress.Progress{Request: event, Client: cli}

	if storage.IsVolumeActive(&volume, &storagePool) {
		return reply(utils.GetResponseData(event), event, cli)
	}

	if err := storage.DoVolumeActivate(&volume, &storagePool, &progress); err != nil {
		return errors.Wrap(err, "Failed to active volume")
	}
	if !storage.IsVolumeActive(&volume, &storagePool) {
		return errors.New("operation failed")
	}
	return reply(utils.GetResponseData(event), event, cli)
}

func VolumeDeactivate(event *revents.Event, cli *client.RancherClient) error {
	var volumeStoragePoolMap model.VolumeStoragePoolMap
	mapstructure.Decode(event.Data["volumeStoragePoolMap"], &volumeStoragePoolMap)
	volume := volumeStoragePoolMap.Volume
	storagePool := volumeStoragePoolMap.StoragePool
	progress := progress.Progress{}

	if storage.IsVolumeInactive(&volume, &storagePool) {
		return reply(utils.GetResponseData(event), event, cli)
	}

	if err := storage.DoVolumeDeactivate(&volume, &storagePool, &progress); err != nil {
		return errors.Wrap(err, "Failed to deactivate volume")
	}
	if !storage.IsVolumeInactive(&volume, &storagePool) {
		return errors.New("operation failed")
	}
	return reply(utils.GetResponseData(event), event, cli)
}

func VolumeRemove(event *revents.Event, cli *client.RancherClient) error {
	var volumeStoragePoolMap model.VolumeStoragePoolMap
	mapstructure.Decode(event.Data["volumeStoragePoolMap"], &volumeStoragePoolMap)
	volume := volumeStoragePoolMap.Volume
	storagePool := volumeStoragePoolMap.StoragePool
	progress := progress.Progress{Request: event, Client: cli}

	if volume.DeviceNumber == 0 {
		compute.PurgeState(volume.Instance)
	}
	if !storage.IsVolumeRemoved(&volume, &storagePool) {
		storage.DoVolumeRemove(&volume, &storagePool, &progress)
	}
	return reply(utils.GetResponseData(event), event, cli)
}
