package storage

import (
	"fmt"
	"path/filepath"

	"os"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/api/types"
	engineCli "github.com/docker/docker/client"
	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	"github.com/rancher/agent/core/progress"
	"github.com/rancher/agent/model"
	"github.com/rancher/agent/utilities/constants"
	"github.com/rancher/agent/utilities/utils"
	"golang.org/x/net/context"
)

const (
	rancherSockDir = "/var/run/rancher/storage"
)

var rancherDrivers = map[string]bool{
	"rancher-ebs":     true,
	"rancher-nfs":     true,
	"rancher-efs":     true,
	"rancher-secrets": true,
}

func VolumeActivateDocker(volume model.Volume, storagePool model.StoragePool, progress *progress.Progress, client *engineCli.Client) error {
	if ok, err := IsVolumeActive(volume, storagePool, client); ok {
		return nil
	} else if err != nil {
		return errors.Wrap(err, constants.VolumeActivateError+"failed to check whether volume is activated")
	}

	if err := DoVolumeActivate(volume, storagePool, progress, client); err != nil {
		return errors.Wrap(err, constants.VolumeActivateError+"failed to activate volume")
	}
	if ok, err := IsVolumeActive(volume, storagePool, client); !ok && err != nil {
		return errors.Wrap(err, constants.VolumeActivateError)
	} else if !ok && err == nil {
		return errors.New(constants.VolumeActivateError + "volume is not activated")
	}
	return nil
}

func VolumeRemoveDocker(volume model.Volume, storagePool model.StoragePool, progress *progress.Progress, dockerClient *engineCli.Client, ca *cache.Cache, resourceID string) error {
	if ok, err := IsVolumeRemoved(volume, storagePool, dockerClient); err == nil && !ok {
		rmErr := DoVolumeRemove(volume, storagePool, progress, dockerClient, ca, resourceID)
		if rmErr != nil {
			return errors.Wrap(rmErr, constants.VolumeRemoveError+"failed to remove volume")
		}
	} else if err != nil {
		return errors.Wrap(err, constants.VolumeRemoveError+"failed to check whether volume is removed")
	}
	return nil
}

func VolumeActivateFlex(volume model.Volume) error {
	payload := struct{ Name string }{Name: volume.Name}
	_, err := CallRancherStorageVolumePlugin(volume, Create, payload)
	if err != nil {
		return err
	}
	return nil
}

func VolumeRemoveFlex(volume model.Volume) error {
	payload := struct{ Name string }{Name: volume.Name}
	_, err := CallRancherStorageVolumePlugin(volume, Remove, payload)
	if err != nil {
		return err
	}
	return nil
}

func DoVolumeActivate(volume model.Volume, storagePool model.StoragePool, progress *progress.Progress, client *engineCli.Client) error {
	if !isManagedVolume(volume) {
		return nil
	}
	driver := volume.Data.Fields.Driver
	driverOpts := volume.Data.Fields.DriverOpts
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
				return errors.Wrap(err, constants.DoVolumeActivateError+"failed to remove volume")
			}
		}
	}

	options := types.VolumeCreateRequest{
		Name:       volume.Name,
		Driver:     driver,
		DriverOpts: opts,
	}
	_, err1 := client.VolumeCreate(context.Background(), options)
	if err1 != nil {
		return errors.Wrap(err1, constants.DoVolumeActivateError+"failed to create volume")
	}
	return nil
}

func DoVolumeRemove(volume model.Volume, storagePool model.StoragePool, progress *progress.Progress, dockerClient *engineCli.Client, ca *cache.Cache, resourceID string) error {
	if _, ok := ca.Get(resourceID); ok {
		ca.Delete(resourceID)
		return nil
	}
	if ok, err := IsVolumeRemoved(volume, storagePool, dockerClient); ok {
		return nil
	} else if err != nil {
		return errors.Wrap(err, constants.DoVolumeRemoveError+"failed to check whether volume is removed")
	}
	if volume.DeviceNumber == 0 {
		container, err := utils.GetContainer(dockerClient, volume.Instance, false)
		if err != nil {
			if !utils.IsContainerNotFoundError(err) {
				return errors.Wrap(err, constants.DoVolumeRemoveError+"faild to get container")
			}
		}
		if container.ID == "" {
			return nil
		}
		errorList := []error{}
		for i := 0; i < 3; i++ {
			if err := utils.RemoveContainer(dockerClient, container.ID); err != nil && !engineCli.IsErrContainerNotFound(err) {
				errorList = append(errorList, err)
			} else {
				break
			}
			time.Sleep(time.Second * 1)
		}
		if len(errorList) == 3 {
			ca.Add(resourceID, true, cache.DefaultExpiration)
			logrus.Warnf("Failed to remove container id [%v]. Tried three times and failed. Error msg: %v", container.ID, errorList)
		}
	} else if isManagedVolume(volume) {
		errorList := []error{}
		for i := 0; i < 3; i++ {
			err := dockerClient.VolumeRemove(context.Background(), volume.Name, false)
			if err != nil {
				if strings.Contains(err.Error(), "Should retry") {
					return errors.Wrap(err, constants.DoVolumeRemoveError+"Error removing volume")
				}
				errorList = append(errorList, err)
			} else {
				break
			}
			time.Sleep(time.Second * 1)
		}
		if len(errorList) == 3 {
			ca.Add(resourceID, true, cache.DefaultExpiration)
			logrus.Warnf("Failed to remove volume name [%v]. Tried three times and failed. Error msg: %v", volume.Name, errorList)
		}
		return nil
	}
	path := pathToVolume(volume)
	if !volume.Data.Fields.IsHostPath {
		_, existErr := os.Stat(path)
		if existErr == nil {
			if err := os.RemoveAll(path); err != nil {
				return errors.Wrap(err, constants.DoVolumeRemoveError+"failed to remove directory")
			}
		}
	}
	return nil
}

func isManagedVolume(volume model.Volume) bool {
	driver := volume.Data.Fields.Driver
	if driver == "" {
		return false
	}
	if _, ok := rancherDrivers[driver]; ok {
		return true
	}
	if volume.Name == "" {
		return false
	}
	return true
}

func pathToVolume(volume model.Volume) string {
	return strings.Replace(volume.URI, "file://", "", -1)
}

func IsVolumeActive(volume model.Volume, storagePool model.StoragePool, dockerClient *engineCli.Client) (bool, error) {
	if !isManagedVolume(volume) {
		return true, nil
	}
	vol, err := dockerClient.VolumeInspect(context.Background(), volume.Name)
	if engineCli.IsErrVolumeNotFound(err) {
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
	if _, ok := rancherDrivers[volume.Data.Fields.Driver]; ok {
		return true
	}
	return false
}

func IsVolumeRemoved(volume model.Volume, storagePool model.StoragePool, client *engineCli.Client) (bool, error) {
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
