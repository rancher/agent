package storage

import (
	"os"

	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/rancher/agent/model"
	"github.com/rancher/agent/utilities/constants"
	"github.com/rancher/agent/utilities/utils"
	"golang.org/x/net/context"
)

func IsVolumeActive(volume model.Volume, storagePool model.StoragePool, dockerClient *client.Client) (bool, error) {
	if !isManagedVolume(volume) {
		return true, nil
	}
	vol, err := dockerClient.VolumeInspect(context.Background(), volume.Name)
	if client.IsErrVolumeNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, errors.Wrap(err, constants.IsVolumeActiveError)
	}
	if vol.Mountpoint != "" {
		return vol.Mountpoint != "moved", nil
	}
	return true, nil
}

func IsImageActive(image model.Image, storagePool model.StoragePool, dockerClient *client.Client) (bool, error) {
	if utils.IsImageNoOp(image) {
		return true, nil
	}
	_, _, err := dockerClient.ImageInspectWithRaw(context.Background(), image.Name)
	if err == nil {
		return true, nil
	} else if client.IsErrImageNotFound(err) {
		return false, nil
	}
	return false, errors.Wrap(err, constants.IsImageActiveError)
}

func IsVolumeRemoved(volume model.Volume, storagePool model.StoragePool, client *client.Client) (bool, error) {
	if volume.DeviceNumber == 0 {
		container, err := utils.GetContainer(client, volume.Instance, false)
		if err != nil {
			if !utils.IsContainerNotFoundError(err) {
				return false, errors.Wrap(err, constants.IsVolumeRemovedError+"failed to get container")
			}
		}
		return container.ID == "", nil
	} else if isManagedVolume(volume) {
		ok, err := IsVolumeActive(volume, storagePool, client)
		if err != nil {
			return false, errors.Wrap(err, constants.IsVolumeRemovedError+"failed to check whether volume is activated")
		}
		return !ok, nil
	}
	path := pathToVolume(volume)
	if !volume.Data.Fields.IsHostPath {
		return true, nil
	}
	_, exist := os.Stat(path)
	if exist != nil {
		return true, nil
	}
	return false, nil

}
