package handlers

import (
	"errors"
	"github.com/mitchellh/mapstructure"
	"github.com/rancher/agent/handlers/progress"
	"github.com/rancher/agent/handlers/utils"
	"github.com/rancher/agent/model"
	revents "github.com/rancher/go-machine-service/events"
	"github.com/rancher/go-rancher/client"
)

func ImageActivate(event *revents.Event, cli *client.RancherClient) error {
	var imageStoragePoolMap model.ImageStoragePoolMap
	mapstructure.Decode(event.Data["imageStoragePoolMap"], &imageStoragePoolMap)
	image := imageStoragePoolMap.Image
	storagePool := imageStoragePoolMap.StoragePool

	progress := progress.Progress{Request: event, Client: cli}

	if image.ID >= 0 && event.Data["processData"] != nil {
		image.ProcessData = event.Data["processData"].(map[string]interface{})
	}
	if utils.IsImageActive(&image, &storagePool) {
		return reply(utils.GetResponseData(event), event, cli)
	}

	if utils.IsImageActive(&image, &storagePool) {
		return reply(event.Data, event, cli)
	}

	err := utils.DoImageActivate(&image, &storagePool, &progress)
	if err != nil {
		return err
	}

	if !utils.IsImageActive(&image, &storagePool) {
		return errors.New("operation failed")
	}

	return reply(event.Data, event, cli)
}

func VolumeActivate(event *revents.Event, cli *client.RancherClient) error {
	var volumeStoragePoolMap model.VolumeStoragePoolMap
	mapstructure.Decode(event.Data["volumeStoragePoolMap"], &volumeStoragePoolMap)
	volume := volumeStoragePoolMap.Volume
	storagePool := volumeStoragePoolMap.StoragePool
	progress := progress.Progress{Request: event, Client: cli}

	if utils.IsVolumeActive(&volume, &storagePool) {
		return reply(utils.GetResponseData(event), event, cli)
	}

	err := utils.DoVolumeActivate(&volume, &storagePool, &progress)
	if err != nil {
		return err
	}
	if !utils.IsVolumeActive(&volume, &storagePool) {
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

	if utils.IsVolumeInactive(&volume, &storagePool) {
		return reply(utils.GetResponseData(event), event, cli)
	}

	err := utils.DoVolumeDeactivate(&volume, &storagePool, &progress)
	if err != nil {
		return err
	}
	if !utils.IsVolumeInactive(&volume, &storagePool) {
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
		utils.PurgeState(volume.Instance)
	}
	if !utils.IsVolumeRemoved(&volume, &storagePool) {
		utils.DoVolumeRemove(&volume, &storagePool, &progress)
	}
	return reply(utils.GetResponseData(event), event, cli)
}
