package storage

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/rancher/agent/core/marshaller"
	"github.com/rancher/agent/core/progress"
	"github.com/rancher/agent/model"
	"github.com/rancher/agent/utilities/config"
	"github.com/rancher/agent/utilities/constants"
	"github.com/rancher/agent/utilities/docker"
	"github.com/rancher/agent/utilities/utils"
	"golang.org/x/net/context"
	"os"
	"strings"
)

func IsVolumeActive(volume *model.Volume, storagePool *model.StoragePool) bool {
	if !isManagedVolume(volume) {
		return true
	}
	version := config.StorageAPIVersion()
	docker.DefaultClient.UpdateClientVersion(version)
	defer docker.DefaultClient.UpdateClientVersion(constants.DefaultVersion)
	vol, err := docker.DefaultClient.VolumeInspect(context.Background(), volume.Name)
	if err != nil {
		logrus.Error(err)
		return false
	}
	if vol.Mountpoint != "" {
		return vol.Mountpoint != "moved"
	}
	return true
}

func isManagedVolume(volume *model.Volume) bool {
	if driver, ok := utils.GetFieldsIfExist(volume.Data, "fields", "driver"); !ok || driver == "" {
		return false
	}
	if volume.Name == "" {
		return false
	}
	return true
}

func imagePull(params *model.ImageParams, progress *progress.Progress) error {
	if !IsImageActive(&params.Image, nil) {
		return DoImageActivate(&params.Image, nil, progress)
	}
	return nil
}

func DoVolumeActivate(volume *model.Volume, storagePool *model.StoragePool, progress *progress.Progress) error {
	if !isManagedVolume(volume) {
		return nil
	}
	driver := utils.InterfaceToString(volume.Data["fields"].(map[string]interface{})["driver"])
	driverOpts := volume.Data["fields"].(map[string]interface{})["driverOpts"]
	opts := map[string]string{}
	if driverOpts != nil {
		driverOpts := driverOpts.(map[string]interface{})
		for k, v := range driverOpts {
			opts[k] = utils.InterfaceToString(v)
		}
	}
	v := config.StorageAPIVersion()
	docker.DefaultClient.UpdateClientVersion(v)
	defer docker.DefaultClient.UpdateClientVersion(constants.DefaultVersion)
	client := docker.DefaultClient

	// Rancher longhorn volumes indicate when they've been moved to a
	// different host. If so, we have to delete before we create
	// to cleanup the reference in docker.

	vol, err := client.VolumeInspect(context.Background(), volume.Name)
	if err != nil {
		logrus.Error(err)
	}
	if vol.Mountpoint == "moved" {
		logrus.Info(fmt.Sprintf("Removing moved volume %s so that it can be re-added.", volume.Name))
		if err := client.VolumeRemove(context.Background(), volume.Name); err != nil {
			logrus.Error(err)
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
		return errors.Wrap(err1, "Failed to activate volume")
	}
	logrus.Info(fmt.Sprintf("volume [%s] created", newVolume.Name))
	return nil
}

func PullImage(image *model.Image, progress *progress.Progress) error {
	return DoImageActivate(image, nil, progress)
}

func DoImageActivate(image *model.Image, storagePool *model.StoragePool, progress *progress.Progress) error {
	if utils.IsNoOp(image.Data) {
		return nil
	}

	if isBuild(image) {
		return imageBuild(image, progress)
	}
	authConfig := map[string]string{}
	rc := image.RegistryCredential
	if rc != nil {
		authConfig["username"] = utils.InterfaceToString(rc["publicValue"])
		if value, ok := utils.GetFieldsIfExist(rc, "data", "fields", "email"); ok {
			authConfig["email"] = utils.InterfaceToString(value)
		}
		authConfig["password"] = utils.InterfaceToString(rc["secretValue"])
		if value, ok := utils.GetFieldsIfExist(rc, "data", "fields", "serverAddress"); ok {
			authConfig["serverAddress"] = utils.InterfaceToString(value)
		}
		if authConfig["serveraddress"] == "https://docker.io" {
			authConfig["serveraddress"] = "https://index.docker.io"
		}
	} else {
		logrus.Info("No Registry credential found. Pulling non-authed")
	}

	client := docker.DefaultClient
	var data model.DockerImage
	mapstructure.Decode(image.Data["dockerImage"], &data)
	temp := data.QualifiedName
	if strings.HasPrefix(temp, "docker.io/") {
		temp = "index." + temp
	}
	/*
			Always pass insecure_registry=True to prevent docker-py
		        from pre-verifying the registry. Let the docker daemon handle
		        the verification of and connection to the registry.
	*/
	var auth types.AuthConfig
	mapstructure.Decode(authConfig, &auth)
	tokenInfo, authErr := client.RegistryLogin(context.Background(), auth)
	if authErr != nil {
		logrus.Error(fmt.Sprintf("Authorization error; %s", authErr))
	}
	if progress == nil {
		_, err2 := client.ImagePull(context.Background(), data.FullName,
			types.ImagePullOptions{
				RegistryAuth: tokenInfo.IdentityToken,
			})
		if err2 != nil {
			return errors.Wrap(err2, fmt.Sprintf("Image [%s] failed to pull",
				data.FullName))
		}
	} else {
		lastMessage := ""
		message := ""
		reader, err := client.ImagePull(context.Background(), data.FullName,
			types.ImagePullOptions{
				RegistryAuth: tokenInfo.IdentityToken,
			})
		if err != nil {
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

func imageBuild(image *model.Image, progress *progress.Progress) error {
	client := docker.DefaultClient
	v, _ := utils.GetFieldsIfExist(image.Data, "fields", "build")
	opts := v.(map[string]interface{})

	if utils.IsStrSet(opts, "context") {
		file, err := utils.DownloadFile(utils.InterfaceToString(opts["context"]), config.Builds(), nil, "")
		if err == nil {
			// delete(opts, "context")
			opts["fileobj"] = file
			opts["custom_context"] = true
			if buildErr := doBuild(opts, progress, client); buildErr != nil {
				return errors.Wrap(buildErr, "Failed to build image")
			}
		}
		if file != "" {
			os.Remove(file)
		}
	} else {
		remote := opts["remote"]
		if strings.HasPrefix(utils.InterfaceToString(remote), "git@github.com:") {
			remote = strings.Replace(utils.InterfaceToString(remote), "git@github.com:", "git://github.com/", -1)
		}
		opts["remote"] = remote
		if buildErr := doBuild(opts, progress, client); buildErr != nil {
			return errors.Wrap(buildErr, "Failed to build image")
		}
	}
	return nil
}

func doBuild(opts map[string]interface{}, progress *progress.Progress, client *client.Client) error {
	remote := utils.InterfaceToString(opts["remote"])
	if remote == "" {
		remote = utils.InterfaceToString(opts["context"])
	}
	logrus.Infof("remote %v, dockerfile %v", remote)
	imageBuildOptions := types.ImageBuildOptions{
		// Dockerfile: dockerFile,
		// for test
		RemoteContext: remote,
		Remove:        true,
		Tags:          []string{utils.InterfaceToString(opts["tag"])},
	}
	response, err := client.ImageBuild(context.Background(), nil, imageBuildOptions)
	if err != nil {
		return errors.Wrap(err, "Failed to build image")
	}
	buffer := utils.ReadBuffer(response.Body)
	statusList := strings.Split(buffer, "\r\n")
	for _, rawStatus := range statusList {
		if rawStatus != "" {
			logrus.Info(rawStatus)
			status := marshaller.FromString(rawStatus)
			if value, ok := utils.GetFieldsIfExist(status, "stream"); ok {
				progress.Update(utils.InterfaceToString(value))
			}
		}
	}
	return nil
}

func isBuild(image *model.Image) bool {
	if build, ok := utils.GetFieldsIfExist(image.Data, "fields", "build"); ok {
		if utils.IsStrSet(build.(map[string]interface{}), "context") ||
			utils.IsStrSet(build.(map[string]interface{}), "remote") {
			return true
		}
	}
	return false
}

func IsImageActive(image *model.Image, storagePool *model.StoragePool) bool {
	if utils.IsNoOp(image.Data) {
		return true
	}
	parsedTag := utils.ParseRepoTag(utils.InterfaceToString(image.Data["dockerImage"].(map[string]interface{})["fullName"]))
	_, _, err := docker.DefaultClient.ImageInspectWithRaw(context.Background(), parsedTag["uuid"], false)
	if err == nil {
		return true
	}
	return false
}

func DoVolumeDeactivate(volume *model.Volume, storagePool *model.StoragePool, progress *progress.Progress) error {
	return errors.New("Not implemented")
}

func IsVolumeInactive(volume *model.Volume, storagePool *model.StoragePool) bool {
	return true
}

func DoVolumeRemove(volume *model.Volume, storagePool *model.StoragePool, progress *progress.Progress) error {
	if IsVolumeRemoved(volume, storagePool) {
		return nil
	}
	if volume.DeviceNumber == 0 {
		client := docker.DefaultClient
		container := utils.GetContainer(client, volume.Instance, false)
		if container == nil {
			return nil
		}
		utils.RemoveContainer(client, container.ID)
	} else if isManagedVolume(volume) {
		version := config.StorageAPIVersion()
		docker.DefaultClient.UpdateClientVersion(version)
		defer docker.DefaultClient.UpdateClientVersion(constants.DefaultVersion)
		err := docker.DefaultClient.VolumeRemove(context.Background(), volume.Name)
		if err != nil {
			if strings.Contains(err.Error(), "409") {
				logrus.Error(fmt.Errorf("Encountered conflict (%s) while deleting volume. Orphaning volume.",
					err.Error()))
			} else {
				return errors.Wrap(err, "Failed to delete volume")
			}
		}
		return nil
	}
	path := pathToVolume(volume)
	if value, ok := utils.GetFieldsIfExist(volume.Data, "fields", "isHostPath"); ok && !utils.InterfaceToBool(value) {
		_, existErr := os.Stat(path)
		if existErr == nil {
			if err := os.RemoveAll(path); err != nil {
				return errors.Wrap(err, "Failed to remove volume")
			}
		}
		return errors.Wrap(existErr, "Failed to remove volume")
	}
	return nil
}

func IsVolumeRemoved(volume *model.Volume, storagePool *model.StoragePool) bool {
	if volume.DeviceNumber == 0 {
		client := docker.DefaultClient
		container := utils.GetContainer(client, volume.Instance, false)
		return container == nil
	} else if isManagedVolume(volume) {
		return !IsVolumeActive(volume, storagePool)
	}
	path := pathToVolume(volume)
	if value, ok := utils.GetFieldsIfExist(volume.Data, "fields", "isHostPath"); ok && utils.InterfaceToBool(value) {
		return true
	}
	_, exist := os.Stat(path)
	if exist != nil {
		return true
	}
	return false

}

func pathToVolume(volume *model.Volume) string {
	return strings.Replace(volume.URI, "file://", "", -1)
}
