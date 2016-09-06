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
	"github.com/rancher/agent/utilities/config"
	"github.com/rancher/agent/utilities/constants"
	"github.com/rancher/agent/utilities/utils"
	"golang.org/x/net/context"
	"os"
	"strings"
)

func IsVolumeActive(volume model.Volume, storagePool model.StoragePool, client *engineCli.Client) (bool, error) {
	if !isManagedVolume(volume) {
		return true, nil
	}
	version := config.StorageAPIVersion()
	client.UpdateClientVersion(version)
	defer client.UpdateClientVersion(constants.DefaultVersion)
	vol, err := client.VolumeInspect(context.Background(), volume.Name)
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

func isManagedVolume(volume model.Volume) bool {
	driver := volume.Data.Fields.Driver
	if driver == "" {
		return false
	}
	if volume.Name == "" {
		return false
	}
	return true
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
	v := config.StorageAPIVersion()
	client.UpdateClientVersion(v)
	defer client.UpdateClientVersion(constants.DefaultVersion)

	// Rancher longhorn volumes indicate when they've been moved to a
	// different host. If so, we have to delete before we create
	// to cleanup the reference in docker.

	vol, err := client.VolumeInspect(context.Background(), volume.Name)
	if err != nil {
		if vol.Mountpoint == "moved" {
			logrus.Info(fmt.Sprintf("Removing moved volume %s so that it can be re-added.", volume.Name))
			if err := client.VolumeRemove(context.Background(), volume.Name, true); err != nil {
				return errors.Wrap(err, constants.DoVolumeActivateError)
			}
		}
	}

	options := types.VolumeCreateRequest{
		Name:       volume.Name,
		Driver:     driver,
		DriverOpts: opts,
	}
	logrus.Infof("start creating volume with options [%+v]", options)
	newVolume, err1 := client.VolumeCreate(context.Background(), options)
	if err1 != nil {
		return errors.Wrap(err1, constants.DoVolumeActivateError)
	}
	logrus.Info(fmt.Sprintf("volume [%s] created", newVolume.Name))
	return nil
}

func PullImage(image model.Image, progress *progress.Progress, client *engineCli.Client) error {
	return DoImageActivate(image, model.StoragePool{}, progress, client)
}

func DoImageActivate(image model.Image, storagePool model.StoragePool, progress *progress.Progress, client *engineCli.Client) error {
	if utils.IsImageNoOp(image.Data) {
		return nil
	}

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
	data := image.Data.DockerImage
	temp := data.QualifiedName
	if strings.HasPrefix(temp, "docker.io/") {
		temp = "index." + temp
	}
	/*
			Always pass insecure_registry=True to prevent docker-py
		        from pre-verifying the registry. Let the docker daemon handle
		        the verification of and connection to the registry.
	*/

	tokenInfo, authErr := client.RegistryLogin(context.Background(), auth)
	if authErr != nil {
		logrus.Error(fmt.Sprintf("Authorization error; %s", authErr))
	}

	if progress == nil {
		_, err := client.ImagePull(context.Background(), data.FullName,
			types.ImagePullOptions{
				RegistryAuth: tokenInfo.IdentityToken,
			})
		if err != nil && !engineCli.IsErrImageNotFound(err) {
			return errors.Wrap(err, fmt.Sprintf("Image [%s] failed to pull",
				data.FullName))
		}
	} else {
		lastMessage := ""
		message := ""
		reader, err := client.ImagePull(context.Background(), data.FullName,
			types.ImagePullOptions{
				RegistryAuth: tokenInfo.IdentityToken,
			})
		if err != nil && !engineCli.IsErrImageNotFound(err) {
			return errors.Wrap(err, "Failed to pull image")
		}
		buffer := utils.ReadBuffer(reader)
		statusList := strings.Split(buffer, "\r\n")
		for _, rawStatus := range statusList {
			if rawStatus != "" {
				status := marshaller.FromString(rawStatus)
				if utils.HasKey(status, "Error") {
					return fmt.Errorf("Image [%s] failed to pull: %s", data.FullName, message)
				}
				if utils.HasKey(status, "status") {
					message = utils.InterfaceToString(status["status"])
				}
			}
		}
		if lastMessage != message {
			progress.Update(message)
			lastMessage = message
		}
	}
	return nil
}

func imageBuild(image model.Image, progress *progress.Progress, client *engineCli.Client) error {
	opts := image.Data.Fields.Build

	if opts.Context != "" {
		file, err := utils.DownloadFile(opts.Context, config.Builds(), nil, "")
		if err == nil {
			opts.FileObj = file
			if buildErr := doBuild(opts, progress, client); buildErr != nil {
				return errors.Wrap(buildErr, constants.ImageBuildError)
			}
		}
		if file != "" {
			// ignore this error because we don't care if that file doesn't exist
			os.Remove(file)
		}
	} else {
		remote := opts.Remote
		if strings.HasPrefix(utils.InterfaceToString(remote), "git@github.com:") {
			remote = strings.Replace(utils.InterfaceToString(remote), "git@github.com:", "git://github.com/", -1)
		}
		opts.Remote = remote
		if buildErr := doBuild(opts, progress, client); buildErr != nil {
			return errors.Wrap(buildErr, constants.ImageBuildError)
		}
	}
	return nil
}

func doBuild(opts model.BuildOptions, progress *progress.Progress, client *engineCli.Client) error {
	remote := opts.Remote
	if remote == "" {
		remote = opts.Context
	}
	imageBuildOptions := types.ImageBuildOptions{
		RemoteContext: remote,
		Remove:        true,
		Tags:          []string{opts.Tag},
	}
	response, err := client.ImageBuild(context.Background(), nil, imageBuildOptions)
	if err != nil {
		return errors.Wrap(err, constants.DoBuildError)
	}
	buffer := utils.ReadBuffer(response.Body)
	statusList := strings.Split(buffer, "\r\n")
	for _, rawStatus := range statusList {
		if rawStatus != "" {
			status := marshaller.FromString(rawStatus)
			if value, ok := utils.GetFieldsIfExist(status, "stream"); ok {
				progress.Update(utils.InterfaceToString(value))
			}
		}
	}
	return nil
}

func isBuild(image model.Image) bool {
	build := image.Data.Fields.Build
	if build.Context != "" || build.Remote != "" {
		return true
	}
	return false
}

func IsImageActive(image model.Image, storagePool model.StoragePool, dockerClient *engineCli.Client) (bool, error) {
	if utils.IsImageNoOp(image.Data) {
		return true, nil
	}
	parsedTag := utils.ParseRepoTag(image.Data.DockerImage.FullName)
	_, _, err := dockerClient.ImageInspectWithRaw(context.Background(), parsedTag["uuid"])
	if err == nil {
		return true, nil
	} else if engineCli.IsErrImageNotFound(err) {
		return false, nil
	}
	return false, errors.Wrap(err, constants.IsImageActiveError)
}

func DoVolumeDeactivate(volume model.Volume, storagePool model.StoragePool, progress *progress.Progress) error {
	return errors.New("Not implemented")
}

func IsVolumeInactive(volume model.Volume, storagePool model.StoragePool) bool {
	return true
}

func DoVolumeRemove(volume model.Volume, storagePool model.StoragePool, progress *progress.Progress, dockerClient *engineCli.Client) error {
	if ok, err := IsVolumeRemoved(volume, storagePool, dockerClient); ok {
		return nil
	} else if err != nil {
		return errors.Wrap(err, constants.DoVolumeRemoveError)
	}
	if volume.DeviceNumber == 0 {
		container, err := utils.GetContainer(dockerClient, volume.Instance, false)
		if err != nil {
			if !utils.IsContainerNotFoundError(err) {
				return errors.Wrap(err, constants.DoVolumeRemoveError)
			}
		}
		if container.ID == "" {
			return nil
		}
		if err := utils.RemoveContainer(dockerClient, container.ID); !engineCli.IsErrContainerNotFound(err) {
			return errors.Wrap(err, constants.DoVolumeRemoveError)
		}
	} else if isManagedVolume(volume) {
		version := config.StorageAPIVersion()
		dockerClient.UpdateClientVersion(version)
		defer dockerClient.UpdateClientVersion(constants.DefaultVersion)
		err := dockerClient.VolumeRemove(context.Background(), volume.Name, false)
		if err != nil {
			if strings.Contains(err.Error(), "409") {
				logrus.Error(fmt.Errorf("Encountered conflict (%s) while deleting volume. Orphaning volume.",
					err.Error()))
			} else {
				return errors.Wrap(err, constants.DoVolumeRemoveError)
			}
		}
		return nil
	}
	path := pathToVolume(volume)
	if !volume.Data.Fields.IsHostPath {
		_, existErr := os.Stat(path)
		if existErr == nil {
			if err := os.RemoveAll(path); err != nil {
				return errors.Wrap(err, constants.DoVolumeRemoveError)
			}
		}
		return errors.Wrap(existErr, constants.DoVolumeRemoveError)
	}
	return nil
}

func IsVolumeRemoved(volume model.Volume, storagePool model.StoragePool, client *engineCli.Client) (bool, error) {
	if volume.DeviceNumber == 0 {
		container, err := utils.GetContainer(client, volume.Instance, false)
		if err != nil {
			if !utils.IsContainerNotFoundError(err) {
				return false, errors.Wrap(err, constants.IsVolumeRemovedError)
			}
		}
		return container.ID == "", nil
	} else if isManagedVolume(volume) {
		ok, err := IsVolumeActive(volume, storagePool, client)
		if err != nil {
			return false, errors.Wrap(err, constants.IsVolumeRemovedError)
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

func pathToVolume(volume model.Volume) string {
	return strings.Replace(volume.URI, "file://", "", -1)
}
