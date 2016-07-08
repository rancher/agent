package handlers

import (
	"golang.org/x/net/context"
	"github.com/Sirupsen/logrus"
	"fmt"
	"strings"
	"errors"
	"github.com/mitchellh/mapstructure"
	"github.com/docker/engine-api/types"
	"os"
	"../handlers/marshaller"
	"github.com/docker/engine-api/client"
)

func is_volume_active(volume Volume) bool {
	if is_managed_volume(volume) {
		return true
	}
	version := storage_api_version()
	vol, err := Get_client(version).VolumeInspect(context.Background(), volume.Name)
	if err != nil {
		return false
	}
	if _, ok := vol["Mountpoint"]; ok {
		return vol["Mountpoint"] != "moved"
	}
	return true
}

func is_managed_volume(volume Volume) bool {
	if driver := volume.Data["field"].(map[string]string)["driver"]; driver == "" {
		return false
	}
	if volume.Name == "" {
		return false
	}
	return true
}

func do_volume_activate(volume Volume){
	if !is_managed_volume(volume) {
		return nil
	}
	driver := volume.Data["field"].(map[string]interface{})["driver"].(string)
	driver_opts := make(map[string]string)
	driver_opts = volume.Data["field"].(map[string]interface{})["driverOpts"].(map[string]string)
	v := storage_api_version()
	client := Get_client(v)

	// Rancher longhorn volumes indicate when they've been moved to a
	// different host. If so, we have to delete before we create
	// to cleanup the reference in docker.

	vol, err := client.VolumeInspect(context.Background(), volume.Name)
	if err != nil {
		logrus.Error(err)
	} else {
		if vol != nil && vol.Mountpoint == "moved" {
			logrus.Info(fmt.Sprintf("Removing moved volume %s so that it can be re-added.", volume.Name))
			client.VolumeRemove(context.Background(), volume.Name)
		}
	}
	options := VolumeCreateRequest{
		Name: volume.Name,
		Driver: driver,
		DriverOpts: driver_opts,
	}
	new_volume, err1 := client.VolumeCreate(context.Background(), options)
	if err1 != nil {
		logrus.Error(err)
	} else {
		logrus.Info(fmt.Sprintf("volume %s created", new_volume.Name))
	}

}

func pull_image(image Image, progress Progress) {
	if !is_image_active(image, nil) {

	}
}

func do_image_activate(image Image, storage_popl interface{}, progress Progress) (error){
	if is_no_op(image) {
		return nil
	}

	if is_build(image) {
		image_build(image, progress)
		return nil
	}
	//TODO why do we need auth_config? for private registry?
	auth_config := map[string]string{}
	rc := image.RegistryCredential
	if rc != nil {
		auth_config["username"] = rc["publicValue"]
		auth_config["email"] = rc["data"].(map[string]interface{})["fields"].
		(map[string]interface{})["email"]
		auth_config["password"] = rc["secretValue"]
		auth_config["serveraddress"] =  rc["registry"].(map[string]interface{})["data"].
		(map[string]interface{})["fields"].(map[string]interface{})["serverAddress"]
		if auth_config["serveraddress"] == "https://docker.io" {
			auth_config["serveraddress"] = "https://index.docker.io"
		}
	} else{
		logrus.Debug("No Registry credential found. Pulling non-authed")
	}

	client := Get_client(DEFAULT_VERSION)
	var data DockerImage
	if err := mapstructure.Decode(image.Data["dockerImage"], &data); err {
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
	var auth AuthConfig
	if err := mapstructure.Decode(auth_config, &auth); err {
		panic(err)
	}
	token_info, auth_err := client.RegistryLogin(context.Background(), auth)
	if auth_err != nil {
		logrus.Error(fmt.Sprintf("Authorization error; %s", auth_err))
	}
	if progress == nil {
		_, err2 := client.ImagePull(context.Background(), data.FullName,
			types.ImagePullOptions{
				RegistryAuth: token_info.IdentityToken,
			})
		if err2 != nil {
			return errors.New(fmt.Sprintf("Image [%s] failed to pull: %s",
				data["fullname"], err2))
		}
	} else {
		last_message := ""
		message := ""
		reader, err := client.ImagePull(context.Background(), data.FullName,
			types.ImagePullOptions{
				RegistryAuth: token_info.IdentityToken,
			})
		if err != nil {
			logrus.Error(err)
		}
		buffer := readBuffer(reader)
		//TODO not sure what response we got from status
		//Attention! status is a json array so we have to alter unmarshaller
		status_list := marshaller.UnmarshalEventList(buffer)
		for _, status := range status_list {
			if has_key(status, "error") {
				return errors.New(fmt.Sprintf("Image [%s] failed to pull: %s", data.FullName, message))
			}
			if has_key(status, "status") {
				message = status["error"]
			}
		}
		if last_message != message {
			progress.update(message)
			last_message = message
		}
	}
	return nil
}

func image_build (image *Image, progress *Progress) {
	client := Get_client(DEFAULT_VERSION)
	opts := image.Data["fields"].(map[string]interface{})["build"].(map[string]interface{})

	if is_str_set(opts, "context") {
		file, err := download_file(opts["context"], builds())
		if err == nil {
			delete(opts, "context")
			opts["fileobj"] = file
			opts["custom_context"] = true
			do_build(opts, progress, &client)
		}
		if file != nil {
			os.Remove(file.Name())
		}
	} else {
		remote := opts["remote"]
		if strings.HasPrefix(remote, "git@github.com:") {
			remote = strings.Replace(remote, "git@github.com:", "git://github.com/", -1)
		}
		delete(opts, "remote")
		opts["path"] = remote
		do_build(opts, progress, client)
	}
}

func do_build(opts map[string]interface{}, progress *Progress, client *client.Client){
	for _, key := range []string{"context", "remote"} {
		if opts[key] != nil {
			delete(opts, key)
		}
	}
	opts["stream"] = true
	opts["rm"] = true
	docker_file := ""
	//TODO check if this logic is correct
	if opts["fileobj"] != nil {
		docker_file = opts["fileobj"].(os.File).Name()
	} else {
		docker_file = opts["path"].(string)
	}
	image_build_options := types.ImageBuildOptions{
		Dockerfile: docker_file,
		Remove: true,
	}
	response, err := client.ImageBuild(context.Background(), nil, image_build_options)
	if err != nil {
		logrus.Error(err)
		fmt.Errorf("error: %s", err)
	}
	buffer := readBuffer(response.Body)
	status_list := marshaller.From_string(buffer)
	for _, status := range status_list {
		progress.update(status["stream"])
	}
}

func is_build(image Image){
	if build, ok := image.Data["fields"].(map[string]interface{})["build"].(map[string]interface{}); ok {
		if
	}
}

func is_image_active(image Image, storage_pool interface{}) bool {
	if is_no_op(image) {
		return true
	}
	parsed_tag := parse_repo_tag(image.Data["dockerImage"].(map[string]interface{})["fullName"])
	inspects, _, err := Get_client(DEFAULT_VERSION).ImageInspectWithRaw(context.Background(), parsed_tag["uuid"], false)
	if err == nil {
		if len(inspects) > 0 {
			return true
		}
	}
	return false
}

func parse_repo_tag(name string) string map[string]string {
	if strings.HasPrefix(name, "docker:") {
		name = name[7:]
	}
	n := strings.LastIndex(name, ":")
	if n < 0 {
		return map[string]string{
			"repo": name[:n],
			"tag": "latest",
			"uuid": name + ":latest",
		}
	}
	tag := name[n+1:]
	if strings.Index(tag, "/") < 0 {
		return map[string]string{
			"repo": name[:n],
			"tag": tag,
			"uuid": name,
		}
	}
	return map[string]string{
		"repo": name,
		"tag": "latest",
		"uuid": name + ":latest",
	}
}
