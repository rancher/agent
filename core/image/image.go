package image

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"

	engineCli "github.com/docker/docker/client"
	"github.com/rancher/agent/utilities/docker"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
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

func PullImage(image model.Image, progress *progress.Progress, client *engineCli.Client, imageUUID string) error {
	dockerCli := docker.GetClient(docker.DefaultVersion)
	return DoImageActivate(image, model.StoragePool{}, progress, dockerCli, imageUUID)
}

func DoImageActivate(image model.Image, storagePool model.StoragePool, progress *progress.Progress, client *engineCli.Client, imageUUID string) error {
	if utils.IsImageNoOp(image) {
		return nil
	}
	imageName := utils.ParseRepoTag(imageUUID)
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

func IsImageActive(image model.Image, storagePool model.StoragePool, dockerClient *client.Client) (bool, error) {
	if utils.IsImageNoOp(image) {
		return true, nil
	}
	_, _, err := dockerClient.ImageInspectWithRaw(context.Background(), image.Name)
	if err == nil {
		return true, nil
	} else if client.IsErrImageNotFound(err) {
		return false, nil
	}
	return false, errors.Wrap(err, constants.IsImageActiveError)
}

func pullImageWrap(client *client.Client, imageUUID string, opts types.ImagePullOptions, progress *progress.Progress, withCredential bool) error {
	if !withCredential {
		opts = types.ImagePullOptions{}
	}

	reader, err := client.ImagePull(context.Background(), imageUUID, opts)
	if err != nil {
		return errors.Wrap(err, "Failed to pull image")
	}
	defer reader.Close()
	return wrapReader(reader, imageUUID, progress)
}

func wrapReader(reader io.ReadCloser, imageUUID string, progress *progress.Progress) error {
	lastMessage := ""
	message := ""
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		status := marshaller.FromString(scanner.Text())
		if utils.HasKey(status, "error") {
			return fmt.Errorf("Image [%s] failed to pull: %v", imageUUID, status["error"])
		}
		if utils.HasKey(status, "status") {
			message = utils.InterfaceToString(status["status"])
		}
		if lastMessage != message && progress != nil {
			progress.Update(message, "yes", nil)
			lastMessage = message
		}
	}
	return nil
}

func wrapAuth(auth types.AuthConfig) string {
	buf, err := json.Marshal(auth)
	if err != nil {
		return ""
	}
	return base64.URLEncoding.EncodeToString(buf)
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
	defer response.Body.Close()
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
