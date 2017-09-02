package handlers

import (
	"github.com/docker/docker/client"
	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	"github.com/rancher/agent/progress"
	"github.com/rancher/agent/utils"
	v3 "github.com/rancher/go-rancher/v3"
	"golang.org/x/net/context"
	"strings"
)

const (
	linkName        = "eth0"
	cniStateBaseDir = "/var/lib/rancher/state/cni"
	UUIDLabel       = "io.rancher.container.uuid"
)

func constructDeploymentSyncReply(containerSpec v3.Container, client *client.Client, ca *cache.Cache, networkKind string, pro *progress.Progress) (v3.DeploymentSyncResponse, error) {
	response := v3.DeploymentSyncResponse{}

	containerID, err := utils.FindContainer(client, containerSpec, false)
	if err != nil && !utils.IsContainerNotFoundError(err) {
		return v3.DeploymentSyncResponse{}, errors.Wrap(err, "failed to get container")
	}

	if containerID == "" {
		status := v3.InstanceStatus{}
		status.InstanceUuid = containerSpec.Uuid
		response.InstanceStatus = []v3.InstanceStatus{status}
		return response, nil
	}

	inspect, err := client.ContainerInspect(context.Background(), containerID)
	if err != nil {
		return v3.DeploymentSyncResponse{}, errors.Wrap(err, "failed to inspect container")
	}
	// converting images
	if strings.HasPrefix(inspect.Image, "sha256:") {
		if err := utils.ReplaceFriendlyImage(ca, client, &inspect); err != nil {
			return v3.DeploymentSyncResponse{}, errors.Wrap(err, "failed to get ip of the container")
		}
	}
	dockerIP, err := getIP(inspect, networkKind, pro)
	if err != nil && !utils.IsNoOp(containerSpec) {
		if running, err2 := isRunning(inspect.ID, client); err2 != nil {
			return v3.DeploymentSyncResponse{}, errors.Wrap(err2, "failed to inspect running container")
		} else if running {
			return v3.DeploymentSyncResponse{}, errors.Wrap(err, "failed to get ip of the container")
		}
	}
	status := v3.InstanceStatus{}
	status.ExternalId = inspect.ID
	dockerInspect, err := utils.StructToMap(inspect)
	if err != nil {
		return v3.DeploymentSyncResponse{}, errors.Wrap(err, "failed to unMarshall docker inspect")
	}
	status.DockerInspect = dockerInspect
	status.InstanceUuid = containerSpec.Uuid
	status.PrimaryIpAddress = dockerIP
	status.State = inspect.State.Status
	response.InstanceStatus = []v3.InstanceStatus{status}

	return response, nil
}

func isRunning(id string, client *client.Client) (bool, error) {
	inspect, err := client.ContainerInspect(context.Background(), id)
	if err != nil {
		return false, err
	}
	return inspect.State.Pid != 0, nil
}
