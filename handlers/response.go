package handlers

import (
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/rancher/agent/progress"
	"github.com/rancher/agent/utils"
	v2 "github.com/rancher/go-rancher/v2"
	"golang.org/x/net/context"
)

const (
	linkName        = "eth0"
	cniStateBaseDir = "/var/lib/rancher/state/cni"
	UUIDLabel       = "io.rancher.container.uuid"
	managedNetwork  = "managed"
)

func constructDeploymentSyncReply(containerSpec v2.Container, client *client.Client, networkKind string, pro *progress.Progress) (interface{}, error) {
	response := v2.DeploymentSyncResponse{}

	containerID, err := utils.FindContainer(client, containerSpec, false)
	if err != nil && !utils.IsContainerNotFoundError(err) {
		return map[string]interface{}{}, errors.Wrap(err, "failed to get container")
	}

	if containerID == "" {
		status := v2.InstanceStatus{}
		status.InstanceUuid = containerSpec.Uuid
		response.InstanceStatus = []v2.InstanceStatus{status}
		return response, nil
	}

	inspect, err := client.ContainerInspect(context.Background(), containerID)
	if err != nil {
		return map[string]interface{}{}, errors.Wrap(err, "failed to inspect container")
	}
	dockerIP, err := getIP(inspect, networkKind, pro)
	if err != nil && !utils.IsNoOp(containerSpec) {
		if running, err2 := isRunning(inspect.ID, client); err2 != nil {
			return nil, errors.Wrap(err2, "failed to inspect running container")
		} else if running {
			return nil, errors.Wrap(err, "failed to get ip of the container")
		}
	}
	status := v2.InstanceStatus{}
	status.ExternalId = inspect.ID
	dockerInspect, err := utils.StructToMap(inspect)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unMarshall docker inspect")
	}
	status.DockerInspect = dockerInspect
	status.InstanceUuid = containerSpec.Uuid
	status.PrimaryIpAddress = dockerIP
	status.State = inspect.State.Status
	response.InstanceStatus = []v2.InstanceStatus{status}

	return response, nil
}

func isRunning(id string, client *client.Client) (bool, error) {
	inspect, err := client.ContainerInspect(context.Background(), id)
	if err != nil {
		return false, err
	}
	return inspect.State.Pid != 0, nil
}
