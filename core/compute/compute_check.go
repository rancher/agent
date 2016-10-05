package compute

import (
	"github.com/docker/engine-api/client"
	"github.com/pkg/errors"
	"github.com/rancher/agent/model"
	"github.com/rancher/agent/utilities/utils"
)

func IsInstanceActive(instance model.Instance, host model.Host, client *client.Client) (bool, error) {
	if utils.IsNoOp(instance.ProcessData) {
		return true, nil
	}

	container, err := utils.GetContainer(client, instance, false)
	if err != nil {
		if utils.IsContainerNotFoundError(err) {
			return false, nil
		}
		return false, errors.WithStack(err)
	}
	return isRunning(client, container)
}

func IsInstanceInactive(instance model.Instance, client *client.Client) (bool, error) {
	if utils.IsNoOp(instance.ProcessData) {
		return true, nil
	}

	container, err := utils.GetContainer(client, instance, false)
	if err != nil {
		if !utils.IsContainerNotFoundError(err) {
			return false, errors.WithStack(err)
		}
	}
	return isStopped(client, container)
}

func IsInstanceRemoved(instance model.Instance, dockerClient *client.Client) (bool, error) {
	con, err := utils.GetContainer(dockerClient, instance, false)
	if err != nil {
		if utils.IsContainerNotFoundError(err) {
			return true, nil
		}
		return false, errors.WithStack(err)
	}
	return con.ID == "", nil
}
