package runtime

import (
	"context"
	"fmt"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/rancher/agent/utils"
	v2 "github.com/rancher/go-rancher/v2"
)

func ContainerStop(containerSpec v2.Container, volumes []v2.Volume, client *client.Client, timeout int) error {
	if err := unmountRancherFlexVolume(volumes); err != nil {
		// ignore the error
		logrus.Error(err)
	}

	t := time.Duration(timeout) * time.Second
	containerID, err := utils.FindContainer(client, containerSpec, false)
	if err != nil {
		return errors.Wrap(err, "failed to get container")
	}
	client.ContainerStop(context.Background(), containerID, &t)
	containerID, err = utils.FindContainer(client, containerSpec, false)
	if err != nil {
		return errors.Wrap(err, "failed to get container")
	}
	if ok, err := isStopped(client, containerID); err != nil {
		return errors.Wrap(err, "failed to check whether container is stopped")
	} else if !ok {
		if killErr := client.ContainerKill(context.Background(), containerID, "KILL"); killErr != nil {
			return errors.Wrap(killErr, "failed to kill container")
		}
	}
	if ok, err := isStopped(client, containerID); err != nil {
		return errors.Wrap(err, "failed to check whether container is stopped")
	} else if !ok {
		return fmt.Errorf("Failed to stop container %v", containerSpec.Uuid)
	}
	logrus.Infof("rancher id [%v]: Container [%v] with docker id [%v] has been stopped", containerSpec.Id, containerSpec.Name, containerID)
	return nil
}

func IsContainerStopped(containerSpec v2.Container, client *client.Client) (bool, error) {
	containerID, err := utils.FindContainer(client, containerSpec, false)
	if err != nil {
		if !utils.IsContainerNotFoundError(err) {
			return false, errors.Wrap(err, "failed to get container")
		}
	}
	return isStopped(client, containerID)
}

func isStopped(client *client.Client, containerID string) (bool, error) {
	ok, err := isRunning(client, containerID)
	if err != nil {
		return false, err
	}
	return !ok, nil
}
