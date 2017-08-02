package runtime

import (
	"fmt"

	"os"
	"path/filepath"
	"strings"

	"github.com/Sirupsen/logrus"
	volumeTypes "github.com/docker/docker/api/types/volume"
	engineCli "github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/rancher/agent/progress"
	"github.com/rancher/agent/utils"
	v2 "github.com/rancher/go-rancher/v2"
	"golang.org/x/net/context"
)

const (
	rancherSockDir = "/var/run/rancher/storage"
)

func DoVolumeActivate(volume v2.Volume, client *engineCli.Client, progress *progress.Progress) error {
	if !isManagedVolume(volume) {
		return nil
	}
	driver := volume.Driver
	driverOpts := volume.DriverOpts
	opts := map[string]string{}
	if driverOpts != nil {
		for k, v := range driverOpts {
			opts[k] = utils.InterfaceToString(v)
		}
	}

	// Rancher longhorn volumes indicate when they've been moved to a
	// different host. If so, we have to delete before we create
	// to cleanup the reference in docker.

	vol, err := client.VolumeInspect(context.Background(), volume.Name)
	if err != nil {
		if vol.Mountpoint == "moved" {
			logrus.Info(fmt.Sprintf("Removing moved volume %s so that it can be re-added.", volume.Name))
			if err := client.VolumeRemove(context.Background(), volume.Name, true); err != nil {
				return errors.Wrap(err, "failed to remove volume")
			}
		}
	}

	progress.Update(fmt.Sprintf("Creating volume %s", volume.Name), "yes", nil)
	options := volumeTypes.VolumesCreateBody{
		Name:       volume.Name,
		Driver:     driver,
		DriverOpts: opts,
	}
	_, err1 := client.VolumeCreate(context.Background(), options)
	if err1 != nil {
		return errors.Wrap(err1, "failed to create volume")
	}
	return nil
}

func isManagedVolume(volume v2.Volume) bool {
	driver := volume.Driver
	if driver == "" {
		return false
	}
	if volume.Name == "" {
		return false
	}
	return true
}

func pathToVolume(volume v2.Volume) string {
	return strings.Replace(volume.Uri, "file://", "", -1)
}

func IsVolumeActive(volume v2.Volume, dockerClient *engineCli.Client) (bool, error) {
	if !isManagedVolume(volume) {
		return true, nil
	}
	vol, err := dockerClient.VolumeInspect(context.Background(), volume.Name)
	if engineCli.IsErrVolumeNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, errors.Wrap(err, "failed to inspect volume")
	}
	if vol.Mountpoint != "" {
		return vol.Mountpoint != "moved", nil
	}
	return true, nil
}

func rancherStorageSockPath(volume v2.Volume) string {
	return filepath.Join(rancherSockDir, volume.Driver+".sock")
}

func IsRancherVolume(volume v2.Volume) bool {
	if !isManagedVolume(volume) {
		return false
	}
	sockFile := rancherStorageSockPath(volume)
	if _, err := os.Stat(sockFile); err == nil {
		return true
	}
	return false
}
