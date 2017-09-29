package runtime

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"

	"github.com/Sirupsen/logrus"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/rancher/agent/progress"
	"github.com/rancher/agent/utils"
	v3 "github.com/rancher/go-rancher/v3"
)

type InstancePull struct {
	Kind  string `json:"kind"`
	Image struct {
		Data struct {
			DockerImage struct {
				FullName string `json:"fullName"`
				Server   string `json:"server"`
			}
		}
		RegistryCredential RegistryCredential
	}
	Mode     string `json:"mode"`
	Complete bool   `json:"complete"`
	Tag      string `json:"tag"`
}

type RegistryCredential struct {
	PublicValue string
	SecretValue string
	Data        struct {
		Fields struct {
			ServerAddress string
		}
	}
}

func DoInstancePull(params PullParams, progress *progress.Progress, dockerClient *client.Client, credential v3.Credential) (types.ImageInspect, error) {
	imageName := params.ImageUUID
	existing, _, err := dockerClient.ImageInspectWithRaw(context.Background(), imageName)
	if err != nil && !client.IsErrImageNotFound(err) {
		return types.ImageInspect{}, errors.Wrap(err, "failed to inspect image")
	}
	if params.Mode == "cached" {
		return existing, nil
	}
	if params.Complete {
		_, err := dockerClient.ImageRemove(context.Background(), fmt.Sprintf("%s%s", imageName, params.Tag), types.ImageRemoveOptions{Force: true})
		if err != nil && !client.IsErrImageNotFound(err) {
			return types.ImageInspect{}, errors.Wrap(err, "failed to remove image")
		}
		return types.ImageInspect{}, nil
	}
	if err := ImagePull(progress, dockerClient, params.ImageUUID, credential); err != nil {
		return types.ImageInspect{}, errors.Wrap(err, "failed to pull image")
	}

	if len(params.Tag) > 0 {
		repoTag := fmt.Sprintf("%s%s", imageName, params.Tag)
		if err := dockerClient.ImageTag(context.Background(), imageName, repoTag); err != nil && !client.IsErrImageNotFound(err) {
			return types.ImageInspect{}, errors.Wrap(err, "failed to tag image")
		}
	}
	inspect, _, err2 := dockerClient.ImageInspectWithRaw(context.Background(), imageName)
	if err2 != nil && !client.IsErrImageNotFound(err) {
		return types.ImageInspect{}, errors.Wrap(err, "failed to inspect image")
	}
	return inspect, nil
}

func ImagePull(progress *progress.Progress, client *client.Client, imageName string, credential v3.Credential) error {
	named, err := reference.ParseNormalizedNamed(imageName)
	if err != nil {
		return errors.Wrap(err, "failed to parse normalized image name")
	}
	auth := types.AuthConfig{
		Username:      credential.PublicValue,
		ServerAddress: reference.Domain(named),
		Password:      credential.SecretValue,
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
		status := utils.FromString(scanner.Text())
		if utils.HasKey(status, "error") {
			return fmt.Errorf("Image [%s] failed to pull: %s", imageUUID, message)
		}
		if utils.HasKey(status, "status") {
			message = fmt.Sprintf("%s: %v", imageUUID, status["status"])
		}
		if lastMessage != message && progress != nil {
			progress.Update(message, "yes", nil)
			lastMessage = message
		}
	}
	logrus.Infof("Docker Image [%v] has been pulled successfully", imageUUID)
	return nil
}

func wrapAuth(auth types.AuthConfig) string {
	buf, err := json.Marshal(auth)
	if err != nil {
		return ""
	}
	return base64.URLEncoding.EncodeToString(buf)
}
