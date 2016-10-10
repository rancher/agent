package storage

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	engineCli "github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/pkg/errors"
	"github.com/rancher/agent/core/marshaller"
	"github.com/rancher/agent/core/progress"
	"github.com/rancher/agent/model"
	"github.com/rancher/agent/utilities/constants"
	"github.com/rancher/agent/utilities/utils"
	"golang.org/x/net/context"
	"os"
	"strings"
)

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

func PullImage(image model.Image, progress *progress.Progress, client *engineCli.Client, imageUUID string) error {
	return DoImageActivate(image, model.StoragePool{}, progress, client, imageUUID)
}

func DoImageActivate(image model.Image, storagePool model.StoragePool, progress *progress.Progress, client *engineCli.Client, imageUUID string) error {
	if utils.IsImageNoOp(image.Data) {
		return nil
	}
	dockerImage := utils.ParseRepoTag(imageUUID)
	realImageUUID := dockerImage.UUID
	if isBuild(image) {
		return imageBuild(image, progress, client)
	}
	rc := image.RegistryCredential
	auth := types.AuthConfig{
		Username:      rc.PublicValue,
		Email:         rc.Data.Fields.Email,
		ServerAddress: rc.Data.Fields.ServerAddress,
		Password:      rc.SecretValue,
	}
	if auth.ServerAddress == "https://docker.io" {
		auth.ServerAddress = "https://index.docker.io"
	}
	cre = rc
	pullOption := types.ImagePullOptions{}
	if rc.PublicValue != "" {
		tokenInfo, authErr := client.RegistryLogin(context.Background(), auth)
		if authErr != nil {
			logrus.Error(fmt.Sprintf("Authorization error; %s", authErr))
		}
		pullOption.RegistryAuth = tokenInfo.IdentityToken
		pullOption.PrivilegeFunc = authFunc
	}

	lastMessage := ""
	message := ""
	reader, err := client.ImagePull(context.Background(), realImageUUID, pullOption)
	if err != nil {
		return errors.Wrap(err, "Failed to pull image")
	}
	defer reader.Close()
	buffer := utils.ReadBuffer(reader)
	statusList := strings.Split(buffer, "\r\n")
	for _, rawStatus := range statusList {
		if rawStatus != "" {
			status := marshaller.FromString(rawStatus)
			if utils.HasKey(status, "error") {
				return fmt.Errorf("Image [%s] failed to pull: %s", realImageUUID, message)
			}
			if utils.HasKey(status, "status") {
				message = utils.InterfaceToString(status["status"])
			}
		}
		if lastMessage != message && progress != nil {
			progress.Update(message, "yes", nil)
			lastMessage = message
		}
	}
	return nil
}

func DoVolumeDeactivate(volume model.Volume, storagePool model.StoragePool, progress *progress.Progress) error {
	return errors.New("Not implemented")
}

func DoVolumeRemove(volume model.Volume, storagePool model.StoragePool, progress *progress.Progress, dockerClient *engineCli.Client) error {
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
		if err := utils.RemoveContainer(dockerClient, container.ID); !engineCli.IsErrContainerNotFound(err) {
			return errors.Wrap(err, constants.DoVolumeRemoveError+"failed to remove container")
		}
	} else if isManagedVolume(volume) {
		err := dockerClient.VolumeRemove(context.Background(), volume.Name, false)
		if err != nil {
			if strings.Contains(err.Error(), "409") {
				logrus.Error(fmt.Errorf("Encountered conflict (%s) while deleting volume. Orphaning volume.",
					err.Error()))
			} else {
				return errors.Wrap(err, constants.DoVolumeRemoveError+"failed to remove volume")
			}
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
		return errors.Wrap(existErr, constants.DoVolumeRemoveError+"failed to find the path")
	}
	return nil
}
