package storage

import (
	"os"

	"github.com/docker/engine-api/client"
	"github.com/pkg/errors"
	"github.com/rancher/agent/model"
	"github.com/rancher/agent/utilities/config"
	"github.com/rancher/agent/utilities/constants"
	"github.com/rancher/agent/utilities/utils"
	"golang.org/x/net/context"
)

func IsVolumeActive(volume model.Volume, storagePool model.StoragePool, dockerClient *client.Client) (bool, error) {
	if !isManagedVolume(volume) {
		return true, nil
	}
	version := config.StorageAPIVersion()
	dockerClient.UpdateClientVersion(version)
	defer dockerClient.UpdateClientVersion(constants.DefaultVersion)
	vol, err := dockerClient.VolumeInspect(context.Background(), volume.Name)
	if client.IsErrVolumeNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, errors.WithStack(err)
	}
	if vol.Mountpoint != "" {
		return vol.Mountpoint != "moved", nil
	}
	return true, nil
}

func IsVolumeInactive(volume model.Volume, storagePool model.StoragePool) bool {
	return true
}

func IsImageActive(image model.Image, storagePool model.StoragePool, dockerClient *client.Client) (bool, error) {
	if utils.IsImageNoOp(image.Data) {
		return true, nil
	}
	_, _, err := dockerClient.ImageInspectWithRaw(context.Background(), image.Name)
	if err == nil {
		return true, nil
	} else if client.IsErrImageNotFound(err) {
		return false, nil
	}
	return false, errors.WithStack(err)
}

func IsVolumeRemoved(volume model.Volume, storagePool model.StoragePool, client *client.Client) (bool, error) {
	if volume.DeviceNumber == 0 {
		container, err := utils.GetContainer(client, volume.Instance, false)
		if err != nil {
			if !utils.IsContainerNotFoundError(err) {
				return false, errors.WithStack(err)
			}
		}
		return container.ID == "", nil
	} else if isManagedVolume(volume) {
		ok, err := IsVolumeActive(volume, storagePool, client)
		if err != nil {
			return false, errors.WithStack(err)
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
