package handlers

import (
	"github.com/Sirupsen/logrus"
	"github.com/mitchellh/mapstructure"
	"github.com/rancher/agent/handlers/progress"
	"github.com/rancher/agent/handlers/utils"
	"github.com/rancher/agent/model"
	revents "github.com/rancher/go-machine-service/events"
	"github.com/rancher/go-rancher/client"
	"sync"
)

func ImageActivate(event *revents.Event, cli *client.RancherClient) error {
	var imageStoragePoolMap model.ImageStoragePoolMap
	mapstructure.Decode(event.Data["imageStoragePoolMap"], &imageStoragePoolMap)
	image := imageStoragePoolMap.Image
	storagePool := imageStoragePoolMap.StoragePool

	progress := progress.Progress{}

	if image.ID >= 0 && event.Data["processData"] != nil {
		image.ProcessData = event.Data["processData"].(map[string]interface{})
	}
	if utils.IsImageActive(&image, &storagePool) {
		return reply(event.Data, &revents.Event{}, cli)
	}

	imageWithLock := ObjWithLock{obj: image, mu: sync.Mutex{}}
	imageWithLock.mu.Lock()
	defer imageWithLock.mu.Unlock()
	im := imageWithLock.obj.(model.Image)
	if utils.IsImageActive(&im, &storagePool) {
		return reply(event.Data, &revents.Event{}, cli)
	}

	err := utils.DoImageActivate(&im, &storagePool, &progress)
	if err != nil {
		logrus.Error(err)
	}

	if !utils.IsImageActive(&im, &storagePool) {
		logrus.Error("operation failed")
	}

	return reply(event.Data, &revents.Event{}, cli)
}

func VolumeActivate(event *revents.Event, cli *client.RancherClient) error {
	var volumeStoragePoolMap model.VolumeStoragePoolMap
	mapstructure.Decode(event.Data["volumeStoragePoolMap"], &volumeStoragePoolMap)
	volume := volumeStoragePoolMap.Volume
	storagePool := volumeStoragePoolMap.StoragePool
	progress := progress.Progress{}

	if utils.IsVolumeActive(&volume, &storagePool) {
		return reply(utils.GetResponseData(event, event.Data), event, cli)
	}

	volumeWithLock := ObjWithLock{obj: volume, mu: sync.Mutex{}}
	volumeWithLock.mu.Lock()
	defer volumeWithLock.mu.Unlock()
	vol := volumeWithLock.obj.(model.Volume)
	err := utils.DoVolumeActivate(&vol, &storagePool, &progress)
	if err != nil {
		logrus.Error(err)
	}
	if !utils.IsVolumeActive(&volume, &storagePool) {
		logrus.Error("operation failed")
	}
	return reply(utils.GetResponseData(event, event.Data), event, cli)
}

func VolumeDeactivate(event *revents.Event, cli *client.RancherClient) error {
	var volumeStoragePoolMap model.VolumeStoragePoolMap
	mapstructure.Decode(event.Data["volumeStoragePoolMap"], &volumeStoragePoolMap)
	volume := volumeStoragePoolMap.Volume
	storagePool := volumeStoragePoolMap.StoragePool
	progress := progress.Progress{}

	if utils.IsVolumeInactive(&volume, &storagePool) {
		return reply(utils.GetResponseData(event, event.Data), event, cli)
	}

	volumeWithLock := ObjWithLock{obj: volume, mu: sync.Mutex{}}
	volumeWithLock.mu.Lock()
	defer volumeWithLock.mu.Unlock()

	vol := volumeWithLock.obj.(model.Volume)
	err := utils.DoVolumeDeactivate(&vol, &storagePool, &progress)
	if err != nil {
		logrus.Error(err)
	}
	if !utils.IsVolumeInactive(&volume, &storagePool) {
		logrus.Error("operation failed")
	}
	return reply(utils.GetResponseData(event, event.Data), event, cli)
}

func VolumeRemove(event *revents.Event, cli *client.RancherClient) error {
	var volumeStoragePoolMap model.VolumeStoragePoolMap
	mapstructure.Decode(event.Data["volumeStoragePoolMap"], &volumeStoragePoolMap)
	volume := volumeStoragePoolMap.Volume
	storagePool := volumeStoragePoolMap.StoragePool
	progress := progress.Progress{}

	volumeWithLock := ObjWithLock{obj: volume, mu: sync.Mutex{}}
	volumeWithLock.mu.Lock()
	defer volumeWithLock.mu.Unlock()
	if volume.DeviceNumber == 0 {
		utils.PurgeState(volume.Instance)
	}
	if !utils.IsVolumeRemoved(&volume, &storagePool) {
		utils.DoVolumeRemove(&volume, &storagePool, &progress)
	}
	return reply(utils.GetResponseData(event, event.Data), event, cli)
}
