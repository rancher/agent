package storage

import (
	"fmt"
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
	"github.com/rancher/agent/utils/constants"
	"github.com/rancher/agent/utils/utils"
	"golang.org/x/net/context"
)

func DoVolumeActivate(volume model.Volume, client *engineCli.Client) error {
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

func RancherStorageVolumeAttach(volume model.Volume) error {
	if err := callRancherStorageVolumeAttach(volume); err != nil {
		return errors.Wrap(err, constants.RancherStorageVolumeAttachError+"failed to attach volume")
	}
	return nil
}

func PullImage(progress *progress.Progress, client *engineCli.Client, imageUUID string, buildOptions model.BuildOptions, rc model.RegistryCredential) error {
	imageName := utils.ParseRepoTag(imageUUID)
	if isBuild(buildOptions) {
		return imageBuild(buildOptions, progress, client)
	}
	auth := types.AuthConfig{
		Username:      rc.PublicValue,
		Email:         rc.Data.Fields.Email,
		ServerAddress: rc.Data.Fields.ServerAddress,
		Password:      rc.SecretValue,
	}
	if auth.ServerAddress == "https://docker.io" {
		auth.ServerAddress = "https://index.docker.io"
	}
	registryAuth := wrapAuth(auth)
	pullOption := types.ImagePullOptions{
		RegistryAuth: registryAuth,
	}
	withCredential := false
	if auth.Username != "" && auth.Password != "" {
		withCredential = true
	}
	// if the first pull is w/o credential, failed directly. If it is w/ credential, then store the error and try pull w/o credential again
	if withCredential {
		if pullErr := pullImageWrap(client, imageName, pullOption, progress, true); pullErr != nil {
			if err := pullImageWrap(client, imageName, pullOption, progress, false); err != nil {
				return pullErr
			}
		}
		return nil
	}
	return pullImageWrap(client, imageName, pullOption, progress, withCredential)
}

func DoVolumeRemove(volume model.Volume, dockerClient *engineCli.Client, ca *cache.Cache, resourceID string) error {
	if _, ok := ca.Get(resourceID); ok {
		ca.Delete(resourceID)
		return nil
	}
	if ok, err := IsVolumeRemoved(volume, dockerClient); ok {
		return nil
	} else if err != nil {
		return errors.Wrap(err, constants.DoVolumeRemoveError+"failed to check whether volume is removed")
	}
	if isManagedVolume(volume) {
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
