package runtime

import (
	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/client"
	v2 "github.com/rancher/go-rancher/v2"
	"github.com/pkg/errors"
	"github.com/rancher/agent/utils"
)

func ContainerRemove(containerSpec v2.Container, dockerClient *client.Client) error {
	containerId, err := utils.FindContainer(dockerClient, containerSpec, false)
	if err != nil {
		if utils.IsContainerNotFoundError(err) {
			return nil
		}
		return errors.Wrap(err, "failed to get container")
	}
	if err := utils.RemoveContainer(dockerClient, containerId); err != nil {
		return errors.Wrap(err, "failed to remove container")
	}
	logrus.Infof("rancher id [%v]: Container [%v] with docker id [%v] has been removed", containerSpec.Id, containerSpec.Name, containerId)
	return nil
}

func IsContainerRemoved(containerSpec v2.Container, dockerClient *client.Client) (bool, error) {
	containerId, err := utils.FindContainer(dockerClient, containerSpec, false)
	if err != nil {
		if utils.IsContainerNotFoundError(err) {
			return true, nil
		}
		return false, errors.Wrap(err, "failed to get container")
	}
	return containerId == "", nil
}
