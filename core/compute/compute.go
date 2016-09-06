package compute

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/container"
	"github.com/docker/engine-api/types/network"
	"github.com/docker/go-connections/nat"
	"github.com/pkg/errors"
	"github.com/rancher/agent/core/marshaller"
	"github.com/rancher/agent/core/progress"
	"github.com/rancher/agent/core/storage"
	"github.com/rancher/agent/model"
	configuration "github.com/rancher/agent/utilities/config"
	"github.com/rancher/agent/utilities/constants"
	"github.com/rancher/agent/utilities/utils"
	"golang.org/x/net/context"
	"io/ioutil"
	urls "net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

func DoInstanceActivate(instance model.Instance, host model.Host, progress *progress.Progress, dockerClient *client.Client, infoData model.InfoData) error {
	if utils.IsNoOp(instance.ProcessData) {
		return nil
	}
	imageTag, err := getImageTag(instance)
	if err != nil {
		return errors.Wrap(err, constants.DoInstanceActivateError)
	}
	name := instance.UUID
	instanceName := instance.Name
	if len(instanceName) > 0 {
		if str := constants.NameRegexCompiler.FindString(instanceName); str != "" {
			id := fmt.Sprintf("r-%s", instanceName)
			_, inspectErr := dockerClient.ContainerInspect(context.Background(), id)
			if inspectErr != nil && client.IsErrContainerNotFound(inspectErr) {
				name = id
			} else if inspectErr != nil {
				return errors.Wrap(inspectErr, constants.DoInstanceActivateError)
			}
		}
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

	utils.AddLabel(&config, constants.UUIDLabel, instance.UUID)

	if len(instanceName) > 0 {
		utils.AddLabel(&config, constants.ContainerNameLabel, instanceName)
	}

	setupPublishPorts(&hostConfig, instance)

	if err := setupDNSSearch(&hostConfig, instance); err != nil {
		return errors.Wrap(err, constants.DoInstanceActivateError)
	}

	setupLinks(&hostConfig, instance)

	setupHostname(&config, instance)

	setupPorts(&config, instance, &hostConfig)

	setupVolumes(&config, instance, &hostConfig, dockerClient, progress)

	if err := setupNetworking(instance, host, &config, &hostConfig, dockerClient); err != nil {
		return errors.Wrap(err, constants.DoInstanceActivateError)
	}

	flagSystemContainer(instance, &config)

	setupProxy(instance, &config)

	setupCattleConfigURL(instance, &config)

	setupFieldsHostConfig(instance.Data.Fields, &hostConfig)

	setupNetworkingConfig(&networkConfig, instance)

	setupDeviceOptions(&hostConfig, instance, infoData)

	setupFieldsConfig(instance.Data.Fields, &config)

	setupLabels(instance.Data.Fields.Labels, &config)

	container, err := utils.GetContainer(dockerClient, instance, false)
	if err != nil {
		if !utils.IsContainerNotFoundError(err) {
			return errors.Wrap(err, constants.DoInstanceActivateError)
		}
	}
	containerID := container.ID
	created := false
	if len(containerID) == 0 {
		newID, err := createContainer(dockerClient, &config, &hostConfig, imageTag, instance, name, progress)
		if err != nil {
			return errors.Wrap(err, constants.DoInstanceActivateError)
		}
		containerID = newID
		created = true
	}

	logrus.Info(fmt.Sprintf("Starting docker container [%v] docker id [%v]", name, containerID))

	if startErr := dockerClient.ContainerStart(context.Background(), containerID, types.ContainerStartOptions{}); startErr != nil {
		if created {
			if err := utils.RemoveContainer(dockerClient, containerID); err != nil {
				return errors.Wrap(err, constants.DoInstanceActivateError)
			}
		}
		return errors.Wrap(startErr, constants.DoInstanceActivateError)
	}

	if err := RecordState(dockerClient, instance, containerID); err != nil {
		return errors.Wrap(err, constants.DoInstanceActivateError)
	}

	return nil
}

func IsInstanceActive(instance model.Instance, host model.Host, client *client.Client) (bool, error) {
	if utils.IsNoOp(instance.ProcessData) {
		return true, nil
	}

	container, err := utils.GetContainer(client, instance, false)
	if err != nil {
		if utils.IsContainerNotFoundError(err) {
			return false, nil
		}
		return false, errors.Wrap(err, constants.IsInstanceActiveError)
	}
	return isRunning(client, container)
}

func isRunning(client *client.Client, container types.Container) (bool, error) {
	if container.ID == "" {
		return false, nil
	}
	inspect, err := client.ContainerInspect(context.Background(), container.ID)
	if err == nil {
		return inspect.State.Running, nil
	}
	return false, err
}

func RecordState(client *client.Client, instance model.Instance, dockerID string) error {
	if dockerID == "" {
		container, err := utils.GetContainer(client, instance, false)
		if err != nil && !utils.IsContainerNotFoundError(err) {
			return errors.Wrap(err, constants.RecordStateError)
		}
		if container.ID != "" {
			dockerID = container.ID
		}
	}

	if dockerID == "" {
		return nil
	}
	contDir := configuration.ContainerStateDir()

	temFilePath := path.Join(contDir, fmt.Sprintf("tmp-%s", dockerID))
	if ok := utils.IsPathExist(temFilePath); ok {
		if err := os.Remove(temFilePath); err != nil {
			return errors.Wrap(err, constants.RecordStateError)
		}
	}

	filePath := path.Join(contDir, dockerID)
	if ok := utils.IsPathExist(temFilePath); ok {
		if err := os.Remove(filePath); err != nil {
			return errors.Wrap(err, constants.RecordStateError)
		}
	}

	if ok := utils.IsPathExist(contDir); !ok {
		mkErr := os.MkdirAll(contDir, 777)
		if mkErr != nil {
			return errors.Wrap(mkErr, constants.RecordStateError)
		}
	}

	data, err := marshaller.ToString(instance)
	if err != nil {
		return errors.Wrap(err, constants.RecordStateError)
	}
	tempFile, err := ioutil.TempFile(contDir, "tmp-")
	if err != nil {
		return errors.Wrap(err, constants.RecordStateError)
	}

	if writeErr := ioutil.WriteFile(tempFile.Name(), data, 0777); writeErr != nil {
		return errors.Wrap(writeErr, constants.RecordStateError)
	}

	if renameErr := os.Rename(tempFile.Name(), filePath); renameErr != nil {
		return errors.Wrap(renameErr, constants.RecordStateError)
	}
	return nil
}

func createContainer(dockerClient *client.Client, config *container.Config, hostConfig *container.HostConfig,
	imageTag string, instance model.Instance, name string, progress *progress.Progress) (string, error) {
	logrus.Info("Creating docker container from config")
	labels := config.Labels
	if labels[constants.PullImageLabels] == "always" {
		params := model.ImageParams{
			Image:    instance.Image,
			Tag:      "",
			Mode:     "all",
			Complete: false,
		}
		_, err := DoInstancePull(params, progress, dockerClient)
		if err != nil {
			return "", errors.Wrap(err, constants.CreateContainerError)
		}
	}
	config.Image = imageTag

	containerResponse, err := dockerClient.ContainerCreate(context.Background(), config, hostConfig, nil, name)
	logrus.Info(fmt.Sprintf("creating container with name %s", name))
	// if image doesn't exist
	if client.IsErrImageNotFound(err) {
		if err := storage.PullImage(instance.Image, progress, dockerClient); err != nil {
			return "", errors.Wrap(err, constants.CreateContainerError)
		}
		containerResponse, err1 := dockerClient.ContainerCreate(context.Background(), config, hostConfig, nil, name)
		if err1 != nil {
			return "", errors.Wrap(err1, constants.CreateContainerError)
		}
		return containerResponse.ID, nil
	} else if err != nil {
		return "", errors.Wrap(err, constants.CreateContainerError)
	}
	return containerResponse.ID, nil
}

func DoInstancePull(params model.ImageParams, progress *progress.Progress, dockerClient *client.Client) (types.ImageInspect, error) {
	dockerImage := params.Image.Data.DockerImage
	existing, _, err := dockerClient.ImageInspectWithRaw(context.Background(), dockerImage.FullName)
	if err != nil && !client.IsErrImageNotFound(err) {
		return types.ImageInspect{}, errors.Wrap(err, constants.DoInstancePullError)
	}
	if params.Mode == "cached" {
		return existing, nil
	}
	if params.Complete {
		_, err := dockerClient.ImageRemove(context.Background(), dockerImage.FullName+params.Tag, types.ImageRemoveOptions{Force: true})
		if err != nil && !client.IsErrImageNotFound(err) {
			return types.ImageInspect{}, errors.Wrap(err, constants.DoInstancePullError)
		}
		return types.ImageInspect{}, nil
	}
	if err := storage.PullImage(params.Image, progress, dockerClient); err != nil {
		return types.ImageInspect{}, errors.Wrap(err, constants.DoInstancePullError)
	}

	if len(params.Tag) > 0 {
		imageInfo := utils.ParseRepoTag(dockerImage.FullName)
		repoTag := fmt.Sprintf("%s:%s", imageInfo["repo"], imageInfo["tag"]+params.Tag)
		if err := dockerClient.ImageTag(context.Background(), dockerImage.FullName, repoTag); err != nil && !client.IsErrImageNotFound(err) {
			return types.ImageInspect{}, errors.Wrap(err, constants.DoInstancePullError)
		}
	}
	inspect, _, err2 := dockerClient.ImageInspectWithRaw(context.Background(), dockerImage.FullName)
	if err2 != nil && !client.IsErrImageNotFound(err) {
		return types.ImageInspect{}, errors.Wrap(err, constants.DoInstancePullError)
	}
	return inspect, nil
}

func getImageTag(instance model.Instance) (string, error) {
	dockerImage := instance.Image.Data.DockerImage
	if dockerImage.FullName == "" {
		return "", errors.New(constants.StartContainerNoImageError)
	}
	return dockerImage.FullName, nil
}

func initializeMaps(config *container.Config, hostConfig *container.HostConfig) {
	config.Labels = make(map[string]string)
	config.Volumes = make(map[string]struct{})
	config.ExposedPorts = make(map[nat.Port]struct{})
	hostConfig.PortBindings = make(map[nat.Port][]nat.PortBinding)
	hostConfig.StorageOpt = make(map[string]string)
	hostConfig.Tmpfs = make(map[string]string)
	hostConfig.Sysctls = make(map[string]string)
}

func setupHostname(config *container.Config, instance model.Instance) {
	config.Hostname = instance.Hostname
}

func setupPorts(config *container.Config, instance model.Instance,
	hostConfig *container.HostConfig) {
	ports := []model.Port{}
	exposedPorts := map[nat.Port]struct{}{}
	bindings := nat.PortMap{}
	if instance.Ports != nil && len(instance.Ports) > 0 {
		for _, port := range instance.Ports {
			ports = append(ports, model.Port{PrivatePort: port.PrivatePort, Protocol: port.Protocol})
			if port.PrivatePort != 0 {
				bind := nat.Port(fmt.Sprintf("%v/%v", port.PrivatePort, port.Protocol))
				bindAddr := port.Data.Fields.BindAddress
				if _, ok := bindings[bind]; !ok {
					bindings[bind] = []nat.PortBinding{
						nat.PortBinding{
							HostIP:   bindAddr,
							HostPort: utils.ConvertPortToString(port.PublicPort),
						},
					}
				} else {
					bindings[bind] = append(bindings[bind], nat.PortBinding{HostIP: bindAddr,
						HostPort: utils.ConvertPortToString(port.PublicPort)})
				}
				exposedPorts[bind] = struct{}{}
			}

		}
	}

	config.ExposedPorts = exposedPorts

	if len(bindings) > 0 {
		hostConfig.PortBindings = bindings
	}
}

func setupVolumes(config *container.Config, instance model.Instance,
	hostConfig *container.HostConfig, client *client.Client, progress *progress.Progress) error {

	volumes := instance.Data.Fields.DataVolumes
	volumesMap := map[string]struct{}{}
	binds := []string{}
	if len(volumes) > 0 {
		for _, volume := range volumes {
			parts := strings.SplitN(volume, ":", 3)
			if len(parts) == 1 {
				volumesMap[parts[0]] = struct{}{}
			} else if len(parts) > 1 {
				volumesMap[parts[1]] = struct{}{}
				mode := ""
				if len(parts) == 3 {
					mode = parts[2]
				} else {
					mode = "rw"
				}
				bind := fmt.Sprintf("%s:%s:%s", parts[0], parts[1], mode)
				binds = append(binds, bind)
			}
		}
		config.Volumes = volumesMap
		hostConfig.Binds = binds
	}

	containers := []string{}
	if vfsList := instance.DataVolumesFromContainers; vfsList != nil {
		for _, vfs := range vfsList {
			container, err := utils.GetContainer(client, (*vfs), false)
			if err != nil {
				if !utils.IsContainerNotFoundError(err) {
					return errors.Wrap(err, constants.SetupVolumesError)
				}
			}
			if container.ID != "" {
				containers = append(containers, container.ID)
			}
		}
		if len(containers) > 0 {
			hostConfig.VolumesFrom = containers
		}
	}

	if vMounts := instance.VolumesFromDataVolumeMounts; len(vMounts) > 0 {
		for _, vMount := range vMounts {
			storagePool := model.StoragePool{}
			if ok, err := storage.IsVolumeActive(vMount, storagePool, client); !ok && err == nil {
				if err := storage.DoVolumeActivate(vMount, storagePool, progress, client); err != nil {
					return errors.Wrap(err, constants.SetupVolumesError)
				}
			} else if err != nil {
				return errors.Wrap(err, constants.SetupVolumesError)
			}
		}
	}
	return nil
}

func flagSystemContainer(instance model.Instance, config *container.Config) {
	if instance.SystemContainer != "" {
		utils.AddLabel(config, constants.SystemLabels, instance.SystemContainer)
	}
}

func setupProxy(instance model.Instance, config *container.Config) {
	if instance.SystemContainer != "" {
		for _, i := range constants.HTTPProxyList {
			config.Env = append(config.Env, fmt.Sprintf("%v=%v", i, os.Getenv(i)))
		}
	}
}

func setupCattleConfigURL(instance model.Instance, config *container.Config) {
	if instance.AgentID == 0 && !utils.HasLabel(instance) {
		return
	}

	utils.AddLabel(config, constants.AgentIDLabel, strconv.Itoa(instance.AgentID))

	url := configuration.URL()
	if len(url) > 0 {
		parsed, err := urls.Parse(url)

		if err != nil {
			logrus.Error(err)
		} else {
			if strings.Contains(parsed.Host, "localhost") {
				port := configuration.APIProxyListenPort()
				utils.AddToEnv(config, map[string]string{
					"CATTLE_AGENT_INSTANCE":    "true",
					"CATTLE_CONFIG_URL_SCHEME": parsed.Scheme,
					"CATTLE_CONFIG_URL_PATH":   parsed.Path,
					"CATTLE_CONFIG_URL_PORT":   strconv.Itoa(port),
				})
			} else {
				utils.AddToEnv(config, map[string]string{
					"CATTLE_CONFIG_URL": url,
					"CATTLE_URL":        url,
				})
			}
		}
	}
}

func setupLegacyCommand(config *container.Config, fields model.InstanceFields, command string) {
	// This can be removed shortly once cattle removes
	// commandArgs
	if len(command) == 0 || len(strings.TrimSpace(command)) == 0 {
		return
	}
	commandArgs := fields.CommandArgs
	commands := []string{}
	parts := strings.Split(command, " ")
	for _, part := range parts {
		commands = append(commands, part)
	}
	if len(commandArgs) > 0 {
		for _, value := range commandArgs {
			commands = append(commands, value)
		}
	}
	if len(commands) > 0 {
		config.Cmd = commands
	}
}

func DoInstanceDeactivate(instance model.Instance, progress *progress.Progress, client *client.Client, timeout int) error {
	if utils.IsNoOp(instance.ProcessData) {
		return nil
	}
	t := time.Duration(timeout) * time.Second
	container, err := utils.GetContainer(client, instance, false)
	if err != nil {
		return errors.Wrap(err, constants.DoInstanceDeactivateError)
	}
	client.ContainerStop(context.Background(), container.ID, &t)
	container, err = utils.GetContainer(client, instance, false)
	if err != nil {
		return errors.Wrap(err, constants.DoInstanceDeactivateError)
	}
	if ok, err := isStopped(client, container); err != nil {
		return errors.Wrap(err, constants.DoInstanceDeactivateError)
	} else if !ok {
		if killErr := client.ContainerKill(context.Background(), container.ID, "KILL"); killErr != nil {
			return errors.Wrap(killErr, constants.DoInstanceDeactivateError)
		}
	}
	if ok, err := isStopped(client, container); err != nil {
		return errors.Wrap(err, constants.DoInstanceDeactivateError)
	} else if !ok {
		return fmt.Errorf("Failed to stop container %v", instance.UUID)
	}
	logrus.Infof("container id %v deactivated", container.ID)
	return nil
}

func isStopped(client *client.Client, container types.Container) (bool, error) {
	ok, err := isRunning(client, container)
	if err != nil {
		return false, err
	}
	return !ok, nil
}

func IsInstanceInactive(instance model.Instance, client *client.Client) (bool, error) {
	if utils.IsNoOp(instance.ProcessData) {
		return true, nil
	}

	container, err := utils.GetContainer(client, instance, false)
	if err != nil {
		if !utils.IsContainerNotFoundError(err) {
			return false, errors.Wrap(err, constants.IsInstanceInactiveError)
		}
	}
	return isStopped(client, container)
}

func DoInstanceForceStop(request model.InstanceForceStop, dockerClient *client.Client) error {
	time := time.Duration(10)
	if stopErr := dockerClient.ContainerStop(context.Background(), request.ID, &time); client.IsErrContainerNotFound(stopErr) {
		logrus.Infof("container id %v not found", request.ID)
		return nil
	} else if stopErr != nil {
		return errors.Wrap(stopErr, constants.DoInstanceForceStopError)
	}
	logrus.Infof("container id %v is forced to be stopped", request.ID)
	return nil
}

func DoInstanceInspect(inspect model.InstanceInspect, dockerClient *client.Client) (types.ContainerJSON, error) {
	containerID := inspect.ID
	containerList, err := dockerClient.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		return types.ContainerJSON{}, errors.Wrap(err, constants.DoInstanceInspectError)
	}
	result, find := utils.FindFirst(containerList, func(c types.Container) bool {
		return utils.IDFilter(containerID, c)
	})
	if !find {
		name := fmt.Sprintf("/%s", inspect.Name)
		if resultWithNameInspect, ok := utils.FindFirst(containerList, func(c types.Container) bool {
			return utils.NameFilter(name, c)
		}); ok {
			result = resultWithNameInspect
			find = true
		}
	}
	if find {
		logrus.Infof("start inspecting container with id [%s]", result.ID)
		inspectResp, err := dockerClient.ContainerInspect(context.Background(), result.ID)
		if err != nil {
			return types.ContainerJSON{}, errors.Wrap(err, constants.DoInstanceInspectError)
		}
		logrus.Infof("container with id [%s] inspected", result.ID)
		return inspectResp, nil
	}
	return types.ContainerJSON{}, fmt.Errorf("container with id [%v] not found", containerID)
}

func IsInstanceRemoved(instance model.Instance, dockerClient *client.Client) (bool, error) {
	con, err := utils.GetContainer(dockerClient, instance, false)
	if err != nil {
		if utils.IsContainerNotFoundError(err) {
			return true, nil
		}
		return false, errors.Wrap(err, constants.IsInstanceRemovedError)
	}
	return con.ID == "", nil
}

func DoInstanceRemove(instance model.Instance, progress *progress.Progress, dockerClient *client.Client) error {
	container, err := utils.GetContainer(dockerClient, instance, false)
	if err != nil {
		if utils.IsContainerNotFoundError(err) {
			return nil
		}
		return errors.Wrap(err, constants.DoInstanceRemoveError)
	}
	if err := utils.RemoveContainer(dockerClient, container.ID); err != nil {
		return errors.Wrap(err, constants.DoInstanceRemoveError)
	}
	return nil
}

func PurgeState(instance model.Instance, client *client.Client) error {
	container, err := utils.GetContainer(client, instance, false)
	if err != nil {
		if !utils.IsContainerNotFoundError(err) {
			return errors.Wrap(err, constants.PurgeStateError)
		}
	}
	if container.ID == "" {
		return nil
	}
	dockerID := container.ID
	contDir := configuration.ContainerStateDir()
	files := []string{path.Join(contDir, "tmp-"+dockerID), path.Join(contDir, dockerID)}
	for _, f := range files {
		if _, err := os.Stat(f); err == nil {
			if rmErr := os.Remove(f); rmErr != nil {
				return errors.Wrap(rmErr, "Failed to purge state")
			}
		}
	}
	return nil
}

// this method convert fields data to fields in configuration
func setupFieldsConfig(fields model.InstanceFields, config *container.Config) {

	// this one is really weird
	commands, ok := fields.Command.([]interface{})
	if !ok {
		setupLegacyCommand(config, fields, utils.InterfaceToString(commands))
	} else {
		cmds := utils.InterfaceToArray(commands)
		for _, cmd := range cmds {
			config.Cmd = append(config.Cmd, utils.InterfaceToString(cmd))
		}
	}

	envs := []string{}
	for k, v := range fields.Environment {
		envs = append(envs, fmt.Sprintf("%v=%v", k, v))
	}
	config.Env = append(config.Env, envs...)

	config.WorkingDir = fields.WorkingDir

	config.Entrypoint = fields.EntryPoint

	config.Tty = fields.Tty

	config.OpenStdin = fields.StdinOpen

	config.Domainname = fields.DomainName

	for k, v := range fields.Labels {
		config.Labels[k] = v
	}

	config.StopSignal = fields.StopSignal
}

func setupNetworkingConfig(networkConfig *network.NetworkingConfig, instance model.Instance) {

}

func setupLabels(labels map[string]string, config *container.Config) {
	for k, v := range labels {
		config.Labels[k] = utils.InterfaceToString(v)
	}
}
