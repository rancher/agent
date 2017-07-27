package runtime

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/rancher/agent/utils"
)

type InstanceInspect struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

func ContainerInspect(inspect InstanceInspect, dockerClient *client.Client) (types.ContainerJSON, error) {
	containerID := inspect.ID
	if containerID != "" {
		// inspect by id
		containerInspect, err := dockerClient.ContainerInspect(context.Background(), containerID)
		if err != nil && !client.IsErrContainerNotFound(err) {
			return types.ContainerJSON{}, errors.Wrap(err, "Failed to inspect container")
		} else if err == nil {
			return containerInspect, nil
		}
	}
	if inspect.Name != "" {
		// inspect by name
		containerList, err := dockerClient.ContainerList(context.Background(), types.ContainerListOptions{All: true})
		if err != nil {
			return types.ContainerJSON{}, errors.Wrap(err, "failed to list containers")
		}
		find := false
		result := types.Container{}
		name := fmt.Sprintf("/%s", inspect.Name)
		if resultWithNameInspect, ok := utils.FindFirst(containerList, func(c types.Container) bool {
			return utils.NameFilter(name, c)
		}); ok {
			result = resultWithNameInspect
			find = true
		}

		if find {
			inspectResp, err := dockerClient.ContainerInspect(context.Background(), result.ID)
			if err != nil && !client.IsErrContainerNotFound(err) {
				return types.ContainerJSON{}, errors.Wrap(err, "failed to inspect container")
			}
			return inspectResp, nil
		}
	}
	return types.ContainerJSON{}, errors.Errorf("container with id [%v] not found", containerID)
}
