package compute

import (
	"fmt"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/rancher/agent/core/progress"
	"github.com/rancher/agent/core/storage"
	"github.com/rancher/agent/model"
	"github.com/rancher/agent/utilities/constants"
	dutils "github.com/rancher/agent/utilities/docker"
	"github.com/rancher/agent/utilities/utils"
	"golang.org/x/net/context"
)

func DoInstanceActivate(instance model.Instance, host model.Host, progress *progress.Progress, dockerClient *client.Client, infoData model.InfoData) error {
	if utils.IsNoOp(instance.ProcessData) {
		return nil
	}
	imageTag, err := getImageTag(instance)
	if err != nil {
		return errors.Wrap(err, constants.DoInstanceActivateError+"failed to get image tag")
	}

	instanceName := instance.Name
	parts := strings.Split(instance.UUID, "-")
	if len(parts) == 0 {
		return errors.Wrap(err, constants.DoInstanceActivateError+"Failed to parse UUID")
	}
	name := fmt.Sprintf("r-%s", instance.UUID)
	if str := constants.NameRegexCompiler.FindString(instanceName); str != "" {
		// container name is valid
		name = fmt.Sprintf("r-%s-%s", instanceName, parts[0])
	}

	config := container.Config{
		OpenStdin: true,
	}
	hostConfig := container.HostConfig{
		PublishAllPorts: false,
		Privileged:      instance.Data.Fields.Privileged,
		ReadonlyRootfs:  instance.Data.Fields.ReadOnly,
	}
	networkConfig := network.NetworkingConfig{}

	initializeMaps(&config, &hostConfig)

	setupLabels(instance.Data.Fields.Labels, &config)

	utils.AddLabel(&config, constants.UUIDLabel, instance.UUID)

	if len(instanceName) > 0 {
		utils.AddLabel(&config, constants.ContainerNameLabel, instanceName)
	}

	setupFieldsHostConfig(instance.Data.Fields, &hostConfig)

	setupFieldsConfig(instance.Data.Fields, &config)

	setupPublishPorts(&hostConfig, instance)

	if err := setupDNSSearch(&hostConfig, instance); err != nil {
		return errors.Wrap(err, constants.DoInstanceActivateError+"failed to set up DNS search")
	}

	setupLinks(&hostConfig, instance)

	setupHostname(&config, instance)

	setupPorts(&config, instance, &hostConfig)

	if err := setupVolumes(&config, instance, &hostConfig, dockerClient, progress); err != nil {
		return errors.Wrap(err, constants.DoInstanceActivateError+"failed to set up volumes")
	}

	if err := setupNetworking(instance, host, &config, &hostConfig, dockerClient, infoData); err != nil {
		return errors.Wrap(err, constants.DoInstanceActivateError+"failed to set up networking")
	}

	setupProxy(instance, &config, getHostEntries())

	setupCattleConfigURL(instance, &config)

	setupNetworkingConfig(&networkConfig, instance)

	setupDeviceOptions(&hostConfig, instance, infoData)

	setupComputeResourceFields(&hostConfig, instance)

	setupHeathConfig(instance.Data.Fields, &config)

	container, err := utils.GetContainer(dockerClient, instance, false)
	if err != nil {
		if !utils.IsContainerNotFoundError(err) {
			return errors.Wrap(err, constants.DoInstanceActivateError+"failed to get container")
		}
	}
	containerID := container.ID
	created := false
	if containerID == "" {
		newID, err := createContainer(dockerClient, &config, &hostConfig, &networkConfig, imageTag, instance, name, progress)
		if err != nil {
			return errors.Wrap(err, constants.DoInstanceActivateError+"failed to create container")
		}
		containerID = newID
		created = true
	}

	startErr := dutils.Serialize(func() error {
		return dockerClient.ContainerStart(context.Background(), containerID, types.ContainerStartOptions{})
	})
	if startErr != nil {
		if created {
			if err := utils.RemoveContainer(dockerClient, containerID); err != nil {
				return errors.Wrap(err, constants.DoInstanceActivateError+"failed to remove container")
			}
		}
		return errors.Wrap(startErr, constants.DoInstanceActivateError+"failed to start container")
	}

	logrus.Infof("rancher id [%v]: Container with docker id [%v] has been started", instance.ID, containerID)
	return nil
}

func DoInstancePull(params model.ImageParams, progress *progress.Progress, dockerClient *client.Client, opts model.BuildOptions, rc model.RegistryCredential) (types.ImageInspect, error) {
	imageName := utils.ParseRepoTag(params.ImageUUID)
	existing, _, err := dockerClient.ImageInspectWithRaw(context.Background(), imageName)
	if err != nil && !client.IsErrImageNotFound(err) {
		return types.ImageInspect{}, errors.Wrap(err, constants.DoInstancePullError+"failed to inspect image")
	}
	if params.Mode == "cached" {
		return existing, nil
	}
	if params.Complete {
		_, err := dockerClient.ImageRemove(context.Background(), fmt.Sprintf("%s%s", imageName, params.Tag), types.ImageRemoveOptions{Force: true})
		if err != nil && !client.IsErrImageNotFound(err) {
			return types.ImageInspect{}, errors.Wrap(err, constants.DoInstancePullError+"failed to remove image")
		}
		return types.ImageInspect{}, nil
	}
	if err := storage.PullImage(progress, dockerClient, params.ImageUUID, opts, rc); err != nil {
		return types.ImageInspect{}, errors.Wrap(err, constants.DoInstancePullError+"failed to pull image")
	}

	if len(params.Tag) > 0 {
		repoTag := fmt.Sprintf("%s%s", imageName, params.Tag)
		if err := dockerClient.ImageTag(context.Background(), imageName, repoTag); err != nil && !client.IsErrImageNotFound(err) {
			return types.ImageInspect{}, errors.Wrap(err, constants.DoInstancePullError+"failed to tag image")
		}
	}
	inspect, _, err2 := dockerClient.ImageInspectWithRaw(context.Background(), imageName)
	if err2 != nil && !client.IsErrImageNotFound(err) {
		return types.ImageInspect{}, errors.Wrap(err, constants.DoInstancePullError+"failed to inspect image")
	}
	return inspect, nil
}

func DoInstanceDeactivate(instance model.Instance, client *client.Client, timeout int) error {
	if utils.IsNoOp(instance.ProcessData) {
		return nil
	}
	t := time.Duration(timeout) * time.Second
	container, err := utils.GetContainer(client, instance, false)
	if err != nil {
		return errors.Wrap(err, constants.DoInstanceDeactivateError+"failed to get container")
	}
	client.ContainerStop(context.Background(), container.ID, &t)
	container, err = utils.GetContainer(client, instance, false)
	if err != nil {
		return errors.Wrap(err, constants.DoInstanceDeactivateError+"failed to get container")
	}
	if ok, err := isStopped(client, container); err != nil {
		return errors.Wrap(err, constants.DoInstanceDeactivateError+"failed to check whether container is stopped")
	} else if !ok {
		if killErr := client.ContainerKill(context.Background(), container.ID, "KILL"); killErr != nil {
			return errors.Wrap(killErr, constants.DoInstanceDeactivateError+"failed to kill container")
		}
	}
	if ok, err := isStopped(client, container); err != nil {
		return errors.Wrap(err, constants.DoInstanceDeactivateError+"failed to check whether container is stopped")
	} else if !ok {
		return fmt.Errorf("Failed to stop container %v", instance.UUID)
	}
	logrus.Infof("rancher id [%v]: Container with docker id [%v] has been deactivated", instance.ID, container.ID)
	return nil
}

func DoInstanceForceStop(request model.InstanceForceStop, dockerClient *client.Client) error {
	time := time.Duration(10)
	if stopErr := dockerClient.ContainerStop(context.Background(), request.ID, &time); client.IsErrContainerNotFound(stopErr) {
		logrus.Infof("container id %v not found", request.ID)
		return nil
	} else if stopErr != nil {
		return errors.Wrap(stopErr, constants.DoInstanceForceStopError+"failed to stop container")
	}
	return nil
}

func DoInstanceInspect(inspect model.InstanceInspect, dockerClient *client.Client) (types.ContainerJSON, error) {
	containerID := inspect.ID
	if containerID != "" {
		// inspect by id
		containerInspect, err := dockerClient.ContainerInspect(context.Background(), containerID)
		if err != nil && !client.IsErrContainerNotFound(err) {
			return types.ContainerJSON{}, errors.Wrap(err, constants.DoInstanceInspectError+"Failed to inspect container")
		} else if err == nil {
			return containerInspect, nil
		}
	}
	if inspect.Name != "" {
		// inspect by name
		containerList, err := dockerClient.ContainerList(context.Background(), types.ContainerListOptions{All: true})
		if err != nil {
			return types.ContainerJSON{}, errors.Wrap(err, constants.DoInstanceInspectError+"failed to list containers")
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
				return types.ContainerJSON{}, errors.Wrap(err, constants.DoInstanceInspectError+"failed to inspect container")
			}
			return inspectResp, nil
		}
	}
	return types.ContainerJSON{}, errors.Errorf("container with id [%v] not found", containerID)
}

func DoInstanceRemove(instance model.Instance, dockerClient *client.Client) error {
	container, err := utils.GetContainer(dockerClient, instance, false)
	if err != nil {
		if utils.IsContainerNotFoundError(err) {
			return nil
		}
		return errors.Wrap(err, constants.DoInstanceRemoveError+"failed to get container")
	}
	if err := utils.RemoveContainer(dockerClient, container.ID); err != nil {
		return errors.Wrap(err, constants.DoInstanceRemoveError+"failed to remove container")
	}
	logrus.Infof("rancher id [%v]: Container with docker id [%v] has been removed", instance.ID, container.ID)
	return nil
}
