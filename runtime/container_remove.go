package runtime

import (
	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/rancher/agent/utils"
	v3 "github.com/rancher/go-rancher/v3"
)

func ContainerRemove(containerSpec v3.Container, dockerClient *client.Client) error {
	containerID, err := utils.FindContainer(dockerClient, containerSpec, false)
	if err != nil {
		if utils.IsContainerNotFoundError(err) {
			return nil
		}
		return errors.Wrap(err, "failed to get container")
	}
	if err := utils.RemoveContainer(dockerClient, containerID); err != nil {
		return errors.Wrap(err, "failed to remove container")
	}
	logrus.Infof("rancher id [%v]: Container [%v] with docker id [%v] has been removed", containerSpec.Id, containerSpec.Name, containerID)
	return nil
}

func IsContainerRemoved(containerSpec v3.Container, dockerClient *client.Client) (bool, error) {
	containerID, err := utils.FindContainer(dockerClient, containerSpec, false)
	if err != nil {
		if utils.IsContainerNotFoundError(err) {
			return true, nil
		}
		return false, errors.Wrap(err, "failed to get container")
	}
	return containerID == "", nil
}
