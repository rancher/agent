package runtime

import (
	"context"
	"fmt"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/client"
	v2 "github.com/rancher/go-rancher/v2"
	"github.com/pkg/errors"
	"github.com/rancher/agent/utils"
)

func ContainerStop(containerSpec v2.Container, volumes []v2.Volume, client *client.Client, timeout int) error {
	if err := unmountRancherFlexVolume(volumes); err != nil {
		// ignore the error
		logrus.Error(err)
	}

	if utils.IsNoOp(containerSpec) {
		return nil
	}
	t := time.Duration(timeout) * time.Second
	containerId, err := utils.FindContainer(client, containerSpec, false)
	if err != nil {
		return errors.Wrap(err, "failed to get container")
	}
	client.ContainerStop(context.Background(), containerId, &t)
	containerId, err = utils.FindContainer(client, containerSpec, false)
	if err != nil {
		return errors.Wrap(err, "failed to get container")
	}
	if ok, err := isStopped(client, containerId); err != nil {
		return errors.Wrap(err, "failed to check whether container is stopped")
	} else if !ok {
		if killErr := client.ContainerKill(context.Background(), containerId, "KILL"); killErr != nil {
			return errors.Wrap(killErr, "failed to kill container")
		}
	}
	if ok, err := isStopped(client, containerId); err != nil {
		return errors.Wrap(err, "failed to check whether container is stopped")
	} else if !ok {
		return fmt.Errorf("Failed to stop container %v", containerSpec.Uuid)
	}
	logrus.Infof("rancher id [%v]: Container [%v] with docker id [%v] has been deactivated", containerSpec.Id, containerSpec.Name, containerId)
	return nil
}

func IsContainerStopped(containerSpec v2.Container, client *client.Client) (bool, error) {
	if utils.IsNoOp(containerSpec) {
		return true, nil
	}

	containerId, err := utils.FindContainer(client, containerSpec, false)
	if err != nil {
		if !utils.IsContainerNotFoundError(err) {
			return false, errors.Wrap(err, "failed to get container")
		}
	}
	return isStopped(client, containerId)
}

func isStopped(client *client.Client, containerId string) (bool, error) {
	ok, err := isRunning(client, containerId)
	if err != nil {
		return false, err
	}
	return !ok, nil
}
