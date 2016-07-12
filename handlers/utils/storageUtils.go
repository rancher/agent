package utils

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/mitchellh/mapstructure"
	"github.com/rancher/agent/handlers/dockerClient"
	"github.com/rancher/agent/handlers/marshaller"
	"github.com/rancher/agent/handlers/progress"
	"github.com/rancher/agent/model"
	"golang.org/x/net/context"
	"os"
	"strings"
)

func isVolumeActive(volume model.Volume) bool {
	if isManagedVolume(volume) {
		return true
	}
	version := storageAPIVersion()
	vol, err := dockerClient.GetClient(version).VolumeInspect(context.Background(), volume.Name)
	if err != nil {
		return false
	}
	if vol.Mountpoint != "" {
		return vol.Mountpoint != "moved"
	}
	return true
}

func isManagedVolume(volume model.Volume) bool {
	if driver := volume.Data["field"].(map[string]string)["driver"]; driver == "" {
		return false
	}
	if volume.Name == "" {
		return false
	}
	return true
}

func imagePull(params *model.ImageParams, progress *progress.Progress) error {
	if !isImageActive(params.Image, nil) {
		return doImageActivate(params.Image, nil, progress)
	}
	return nil
}

func doVolumeActivate(volume model.Volume) {
	if !isManagedVolume(volume) {
		return
	}
	driver := volume.Data["field"].(map[string]interface{})["driver"].(string)
	driverOpts := make(map[string]string)
	driverOpts = volume.Data["field"].(map[string]interface{})["driverOpts"].(map[string]string)
	v := storageAPIVersion()
	client := dockerClient.GetClient(v)

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
	newVolume, err1 := client.VolumeCreate(context.Background(), options)
	if err1 != nil {
		logrus.Error(err)
	} else {
		logrus.Info(fmt.Sprintf("volume %s created", newVolume.Name))
	}
}

func pullImage(image model.Image, progress *progress.Progress) {
	if !isImageActive(image, nil) {
		doImageActivate(image, nil, progress)
	}
}

//TODO what is a storage pool?
func doImageActivate(image model.Image, storagePool interface{}, progress *progress.Progress) error {
	if isNoOp(image.Data) {
		return nil
	}

	if isBuild(image) {
		imageBuild(&image, progress)
		return nil
	}
	//TODO why do we need authConfig? for private registry?
	authConfig := map[string]string{}
	rc := image.RegistryCredential
	if rc != nil {
		authConfig["username"] = rc["publicValue"].(string)
		if value, ok := getFieldsIfExist(rc, "data", "fields", "email"); ok {
			authConfig["email"] = value.(string)
		}
		authConfig["password"] = rc["secretValue"].(string)
		if value, ok := getFieldsIfExist(rc, "data", "fields", "serverAddress"); ok {
			authConfig["serverAddress"] = value.(string)
		}
		if authConfig["serveraddress"] == "https://docker.io" {
			authConfig["serveraddress"] = "https://index.docker.io"
		}
	} else {
		logrus.Debug("No Registry credential found. Pulling non-authed")
	}

	client := dockerClient.GetClient(DefaultVersion)
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
		buffer := readBuffer(reader)
		//TODO not sure what response we got from status
		//Attention! status is a json array so we have to alter unmarshaller
		statusList := marshaller.UnmarshalEventList([]byte(buffer))
		for _, status := range statusList {
			if hasKey(status, "error") {
				return fmt.Errorf("Image [%s] failed to pull: %s", data.FullName, message)
			}
			if hasKey(status, "status") {
				message = status["error"].(string)
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
	client := dockerClient.GetClient(DefaultVersion)
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
	buffer := readBuffer(response.Body)
	statusList := marshaller.FromString(buffer)
	for _, status := range statusList {
		status := status.(map[string]interface{})
		progress.Update(status["stream"].(string))
	}
}

func isBuild(image model.Image) bool {
	if build, ok := getFieldsIfExist(image.Data, "field", "build"); ok {
		if isStrSet(build.(map[string]interface{}), "context") ||
			isStrSet(build.(map[string]interface{}), "remote") {
			return true
		}
	}
	return false
}

func isImageActive(image model.Image, storagePool interface{}) bool {
	if isNoOp(image.Data) {
		return true
	}
	parsedTag := parseRepoTag(image.Data["dockerImage"].(map[string]interface{})["fullName"].(string))
	_, _, err := dockerClient.GetClient(DefaultVersion).ImageInspectWithRaw(context.Background(), parsedTag["uuid"], false)
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
