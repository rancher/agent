package utils

import (
	"errors"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/mitchellh/mapstructure"
	"github.com/rancher/agent/handlers/docker"
	"github.com/rancher/agent/handlers/marshaller"
	"github.com/rancher/agent/handlers/progress"
	"github.com/rancher/agent/model"
	"golang.org/x/net/context"
	"os"
	"strconv"
	"strings"
)

func IsVolumeActive(volume *model.Volume, storagePool *model.StoragePool) bool {
	if isManagedVolume(volume) {
		return true
	}
	version := storageAPIVersion()
	vol, err := docker.GetClient(version).VolumeInspect(context.Background(), volume.Name)
	if err != nil {
		return false
	}
	if vol.Mountpoint != "" {
		return vol.Mountpoint != "moved"
	}
	return true
}

func isManagedVolume(volume *model.Volume) bool {
	if driver, ok := GetFieldsIfExist(volume.Data, "fields", "driver"); !ok || driver == "" {
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
	driver := volume.Data["field"].(map[string]interface{})["driver"].(string)
	driverOpts := make(map[string]string)
	driverOpts = volume.Data["field"].(map[string]interface{})["driverOpts"].(map[string]string)
	v := storageAPIVersion()
	client := docker.GetClient(v)

	// Rancher longhorn volumes indicate when they've been moved to a
	// different host. If so, we have to delete before we create
	// to cleanup the reference in docker.

	vol, err := client.VolumeInspect(context.Background(), volume.Name)
	if err != nil {
		logrus.Error(err)
	} else {
		if vol.Mountpoint == "moved" {
			logrus.Info(fmt.Sprintf("Removing moved volume %s so that it can be re-added.", volume.Name))
			client.VolumeRemove(context.Background(), volume.Name)
		}
	}
	options := types.VolumeCreateRequest{
		Name:       volume.Name,
		Driver:     driver,
		DriverOpts: driverOpts,
	}
	logrus.Infof("start creating volume name[%v]", options.Name)
	newVolume, err1 := client.VolumeCreate(context.Background(), options)
	if err1 != nil {
		logrus.Error(err)
		return err
	}
	logrus.Info(fmt.Sprintf("volume [%s] created", newVolume.Name))
	return nil
}

func pullImage(image *model.Image, progress *progress.Progress) {
	if !IsImageActive(image, nil) {
		DoImageActivate(image, nil, progress)
	}
}

func DoImageActivate(image *model.Image, storagePool *model.StoragePool, progress *progress.Progress) error {
	if isNoOp(image.Data) {
		return nil
	}

	if isBuild(image) {
		imageBuild(image, progress)
		return nil
	}
	//TODO why do we need authConfig? for private registry?
	authConfig := map[string]string{}
	rc := image.RegistryCredential
	if rc != nil {
		authConfig["username"] = rc["publicValue"].(string)
		if value, ok := GetFieldsIfExist(rc, "data", "fields", "email"); ok {
			authConfig["email"] = value.(string)
		}
		authConfig["password"] = rc["secretValue"].(string)
		if value, ok := GetFieldsIfExist(rc, "data", "fields", "serverAddress"); ok {
			authConfig["serverAddress"] = value.(string)
		}
		if authConfig["serveraddress"] == "https://docker.io" {
			authConfig["serveraddress"] = "https://index.docker.io"
		}
	} else {
		logrus.Debug("No Registry credential found. Pulling non-authed")
	}

	client := docker.GetClient(DefaultVersion)
	var data model.DockerImage
	if err := mapstructure.Decode(image.Data["dockerImage"], &data); err != nil {
		panic(err)
	}
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
	if err := mapstructure.Decode(authConfig, &auth); err != nil {
		panic(err)
	}
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
			return fmt.Errorf("Image [%s] failed to pull: %s",
				data.FullName, err2)
		}
	} else {
		lastMessage := ""
		message := ""
		reader, err := client.ImagePull(context.Background(), data.FullName,
			types.ImagePullOptions{
				RegistryAuth: tokenInfo.IdentityToken,
			})
		if err != nil {
			logrus.Error(err)
		}
		buffer := ReadBuffer(reader)
		//TODO not sure what response we got from status
		//Attention! status is a json array so we have to alter unmarshaller
		logrus.Infof("status data from pull image %s", buffer)
		statusList := strings.Split(buffer, "\r\n")
		for _, rawStatus := range statusList {
			status := marshaller.FromString(rawStatus)
			if hasKey(status, "error") {
				return fmt.Errorf("Image [%s] failed to pull: %s", data.FullName, message)
			}
			if hasKey(status, "status") {
				logrus.Infof("pull image status %s", status["status"].(string))
				message = status["status"].(string)
			}
		}
		if lastMessage != message {
			progress.Update(message)
			lastMessage = message
		}
	}
	return nil
}

func imageBuild(image *model.Image, progress *progress.Progress) {
	client := docker.GetClient(DefaultVersion)
	opts := image.Data["fields"].(map[string]interface{})["build"].(map[string]interface{})

	if isStrSet(opts, "context") {
		file, err := downloadFile(opts["context"].(string), builds(), nil, "")
		if err == nil {
			delete(opts, "context")
			opts["fileobj"] = file
			opts["custom_context"] = true
			doBuild(opts, progress, client)
		}
		if file != "" {
			os.Remove(file)
		}
	} else {
		remote := opts["remote"]
		if strings.HasPrefix(remote.(string), "git@github.com:") {
			remote = strings.Replace(remote.(string), "git@github.com:", "git://github.com/", -1)
		}
		delete(opts, "remote")
		opts["path"] = remote
		doBuild(opts, progress, client)
	}
}

func doBuild(opts map[string]interface{}, progress *progress.Progress, client *client.Client) {
	for _, key := range []string{"context", "remote"} {
		if opts[key] != nil {
			delete(opts, key)
		}
	}
	opts["stream"] = true
	opts["rm"] = true
	dockerFile := ""
	//TODO check if this logic is correct
	if opts["fileobj"] != nil {
		dockerFile = opts["fileobj"].(string)
	} else {
		dockerFile = opts["path"].(string)
	}
	imageBuildOptions := types.ImageBuildOptions{
		Dockerfile: dockerFile,
		Remove:     true,
	}
	response, err := client.ImageBuild(context.Background(), nil, imageBuildOptions)
	if err != nil {
		logrus.Error(err)
	}
	buffer := ReadBuffer(response.Body)
	statusList := marshaller.FromString(buffer)
	for _, status := range statusList {
		status := status.(map[string]interface{})
		progress.Update(status["stream"].(string))
	}
}

func isBuild(image *model.Image) bool {
	if build, ok := GetFieldsIfExist(image.Data, "field", "build"); ok {
		if isStrSet(build.(map[string]interface{}), "context") ||
			isStrSet(build.(map[string]interface{}), "remote") {
			return true
		}
	}
	return false
}

func IsImageActive(image *model.Image, storagePool *model.StoragePool) bool {
	if isNoOp(image.Data) {
		return true
	}
	parsedTag := parseRepoTag(image.Data["dockerImage"].(map[string]interface{})["fullName"].(string))
	_, _, err := docker.GetClient(DefaultVersion).ImageInspectWithRaw(context.Background(), parsedTag["uuid"], false)
	if err == nil {
		return true
	}
	return false
}

func parseRepoTag(name string) map[string]string {
	if strings.HasPrefix(name, "docker:") {
		name = name[7:]
	}
	n := strings.LastIndex(name, ":")
	if n < 0 {
		return map[string]string{
			"repo": name[:n],
			"tag":  "latest",
			"uuid": name + ":latest",
		}
	}
	tag := name[n+1:]
	if strings.Index(tag, "/") < 0 {
		return map[string]string{
			"repo": name[:n],
			"tag":  tag,
			"uuid": name,
		}
	}
	return map[string]string{
		"repo": name,
		"tag":  "latest",
		"uuid": name + ":latest",
	}
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
		client := docker.GetClient(DefaultVersion)
		container := GetContainer(client, volume.Instance, false)
		if container == nil {
			return nil
		}
		removeContainer(client, container.ID)
	} else if isManagedVolume(volume) {
		version := storageAPIVersion()
		err := docker.GetClient(version).VolumeRemove(context.Background(), strconv.Itoa(volume.ID))
		if err != nil {
			if strings.Contains(err.Error(), "409") {
				logrus.Error(fmt.Errorf("Encountered conflict (%s) while deleting volume. Orphaning volume.",
					err.Error()))
			}
			return err
		}
		return nil
	}
	path := pathToVolume(volume)
	var err error
	if value, ok := GetFieldsIfExist(volume.Data, "fields", "isHostPath"); ok && !value.(bool) {
		_, existErr := os.Stat(path)
		if existErr == nil {
			err = os.RemoveAll(path)
			if err != nil {
				logrus.Error(err)
			}
		}
	}
	return err
}

func IsVolumeRemoved(volume *model.Volume, storagePool *model.StoragePool) bool {
	if volume.DeviceNumber == 0 {
		client := docker.GetClient(DefaultVersion)
		container := GetContainer(client, volume.Instance, false)
		return container == nil
	} else if isManagedVolume(volume) {
		return IsVolumeActive(volume, storagePool)
	}
	path := pathToVolume(volume)
	if value, ok := GetFieldsIfExist(volume.Data, "fields", "isHostPath"); ok && value.(bool) {
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
