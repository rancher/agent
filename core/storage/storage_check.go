package storage

import (
	"os"

	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/rancher/agent/model"
	"github.com/rancher/agent/utils/constants"
	"golang.org/x/net/context"
	"path/filepath"
)

const (
	rancherSockDir = "/var/run/rancher/storage"
)

func IsVolumeActive(volume model.Volume, dockerClient *client.Client) (bool, error) {
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

func rancherStorageSockPath(volume model.Volume) string {
	return filepath.Join(rancherSockDir, volume.Data.Fields.Driver+".sock")
}

func IsRancherVolume(volume model.Volume) bool {
	if !isManagedVolume(volume) {
		return false
	}
	sockFile := rancherStorageSockPath(volume)
	if _, err := os.Stat(sockFile); err == nil {
		return true
	}
	return false
}

func IsVolumeRemoved(volume model.Volume, client *client.Client) (bool, error) {
	if isManagedVolume(volume) {
		ok, err := IsVolumeActive(volume, client)
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
