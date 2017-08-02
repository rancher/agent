package runtime

import (
	"os"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/rancher/agent/progress"
	v2 "github.com/rancher/go-rancher/v2"
	"golang.org/x/net/context"
)

func VolumeRemoveDocker(volume v2.Volume, dockerClient *client.Client, pro *progress.Progress) error {
	if ok, err := IsVolumeRemoved(volume, dockerClient); err == nil && !ok {
		rmErr := VolumeRemove(volume, dockerClient, pro)
		if rmErr != nil {
			return errors.Wrap(rmErr, "failed to remove volume")
		}
	} else if err != nil {
		return errors.Wrap(err, "failed to check whether volume is removed")
	}
	return nil
}

func VolumeRemoveFlex(volume v2.Volume) error {
	payload := struct{ Name string }{Name: volume.Name}
	_, err := callRancherStorageVolumePlugin(volume, Remove, payload)
	if err != nil {
		return err
	}
	return nil
}

func VolumeRemove(volume v2.Volume, dockerClient *client.Client, pro *progress.Progress) error {
	if ok, err := IsVolumeRemoved(volume, dockerClient); ok {
		return nil
	} else if err != nil {
		return errors.Wrap(err, "failed to check whether volume is removed")
	}
	errorList := []error{}
	for i := 0; i < 3; i++ {
		pro.Update("Removing volume", "yes", nil)
		err := dockerClient.VolumeRemove(context.Background(), volume.Name, false)
		if err != nil {
			if strings.Contains(err.Error(), "Should retry") {
				return errors.Wrap(err, "Error removing volume")
			}
			errorList = append(errorList, err)
		} else {
			break
		}
		time.Sleep(time.Second * 1)
	}
	if len(errorList) == 3 {
		logrus.Warnf("Failed to remove volume name [%v]. Tried three times and failed. Error msg: %v", volume.Name, errorList)
	}
	path := pathToVolume(volume)
	if !volume.IsHostPath {
		_, existErr := os.Stat(path)
		if existErr == nil {
			if err := os.RemoveAll(path); err != nil {
				return errors.Wrap(err, "failed to remove directory")
			}
		}
	}
	return nil
}

func IsVolumeRemoved(volume v2.Volume, client *client.Client) (bool, error) {
	if isManagedVolume(volume) {
		ok, err := IsVolumeActive(volume, client)
		if err != nil {
			return false, errors.Wrap(err, "failed to check whether volume is activated")
		}
		return !ok, nil
	}
	path := pathToVolume(volume)
	if !volume.IsHostPath {
		return true, nil
	}
	_, exist := os.Stat(path)
	if exist != nil {
		return true, nil
	}
	return false, nil
}
