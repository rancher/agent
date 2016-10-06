package storage

import (
	"encoding/base64"
	"fmt"
	"github.com/docker/engine-api/client"
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

var cre model.RegistryCredential

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

func imageBuild(image model.Image, progress *progress.Progress, dockerClient *client.Client) error {
	opts := image.Data.Fields.Build

	if opts.Context != "" {
		file, err := utils.DownloadFile(opts.Context, config.Builds(), nil, "")
		if err == nil {
			opts.FileObj = file
			if buildErr := doBuild(opts, progress, dockerClient); buildErr != nil {
				return errors.Wrap(buildErr, constants.ImageBuildError+"failed to build image")
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
		if buildErr := doBuild(opts, progress, dockerClient); buildErr != nil {
			return errors.Wrap(buildErr, constants.ImageBuildError+"failed to build image")
		}
	}
	return nil
}

func doBuild(opts model.BuildOptions, progress *progress.Progress, client *client.Client) error {
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
		return errors.Wrap(err, constants.DoBuildError+"failed to build image")
	}
	buffer := utils.ReadBuffer(response.Body)
	statusList := strings.Split(buffer, "\r\n")
	for _, rawStatus := range statusList {
		if rawStatus != "" {
			status := marshaller.FromString(rawStatus)
			if value, ok := utils.GetFieldsIfExist(status, "stream"); ok {
				progress.Update(utils.InterfaceToString(value), "yes", nil)
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

func pathToVolume(volume model.Volume) string {
	return strings.Replace(volume.URI, "file://", "", -1)
}

func authFunc() (string, error) {
	username := cre.PublicValue
	password := cre.SecretValue
	email := cre.Data.Fields.Email
	auth := fmt.Sprintf("{ \"username\": \"%s\", \"password\": \"%s\", \"email\": \"%s\" }", username, password, email)
	str := base64.StdEncoding.EncodeToString([]byte(auth))
	return str, nil
}
