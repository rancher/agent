package compute

import (
	"bufio"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/docker/engine-api/client"
	engineCli "github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/blkiodev"
	"github.com/docker/engine-api/types/container"
	"github.com/docker/engine-api/types/network"
	"github.com/docker/go-connections/nat"
	"github.com/docker/go-units"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/rancher/agent/core/hostInfo"
	"github.com/rancher/agent/core/marshaller"
	"github.com/rancher/agent/core/progress"
	"github.com/rancher/agent/core/storage"
	"github.com/rancher/agent/model"
	configuration "github.com/rancher/agent/utilities/config"
	"github.com/rancher/agent/utilities/constants"
	"github.com/rancher/agent/utilities/docker"
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

func DoInstanceActivate(instance *model.Instance, host *model.Host, progress *progress.Progress) error {
	if utils.IsNoOp(instance.Data) {
		return nil
	}
	dockerClient := docker.DefaultClient

	imageTag, err := getImageTag(instance)
	if err != nil {
		return errors.Wrap(err, "Failed to activate instance")
	}
	name := instance.UUID
	instanceName := instance.Name
	if len(instanceName) > 0 {
		if str := constants.RegexCompiler.FindString(instanceName); str != "" {
			id := fmt.Sprintf("r-%s", instanceName)
			_, inspectErr := dockerClient.ContainerInspect(context.Background(), id)
			if inspectErr != nil && client.IsErrNotFound(inspectErr) {
				name = id
			} else if inspectErr != nil {
				return errors.Wrap(inspectErr, "Failed to activate instance")
			}
		}
	}
	config := container.Config{}
	hostConfig := container.HostConfig{
		PublishAllPorts: false,
		Privileged:      utils.IsTrue(instance, "privileged"),
		ReadonlyRootfs:  utils.IsTrue(instance, "readOnly"),
	}
	networkConfig := network.NetworkingConfig{}

	initializeMaps(&config, &hostConfig)

	utils.AddLabel(&config, constants.UUIDLabel, instance.UUID)

	if len(instanceName) > 0 {
		utils.AddLabel(&config, "io.rancher.container.name", instanceName)
	}

	setupPublishPorts(&hostConfig, instance)

	setupDNSSearch(&hostConfig, instance)

	setupHostname(&config, instance)

	setupPorts(&config, instance, &hostConfig)

	setupVolumes(&config, instance, &hostConfig, dockerClient)

	setupLinks(&hostConfig, instance)

	setupNetworking(instance, host, &config, &hostConfig)

	flagSystemContainer(instance, &config)

	setupProxy(instance, &config)

	setupCattleConfigURL(instance, &config)

	setupResource(instance.Data["fields"].(map[string]interface{}), &hostConfig)

	setupFieldsHostConfig(instance.Data["fields"].(map[string]interface{}), &hostConfig)

	setupNetworkingConfig(&networkConfig, instance)

	setupDeviceOptions(&hostConfig, instance)

	setupFieldsConfig(instance.Data["fields"].(map[string]interface{}), &config)

	if labels, ok := utils.GetFieldsIfExist(instance.Data, "fields", "labels"); ok {
		setupLabels(labels.(map[string]interface{}), &config)
	}

	container := utils.GetContainer(dockerClient, instance, false)
	containerID := ""
	if container != nil {
		containerID = container.ID
	}
	created := false
	if len(containerID) == 0 {
		newID, err := createContainer(dockerClient, &config, &hostConfig, imageTag, instance, name, progress)
		if err != nil {
			return errors.Wrap(err, "Failed to activate instance")
		}
		containerID = newID
		created = true
	}

	logrus.Info(fmt.Sprintf("Starting docker container [%v] docker id [%v]", name, containerID))

	if startErr := dockerClient.ContainerStart(context.Background(), containerID, types.ContainerStartOptions{}); startErr != nil {
		if created {
			if err := utils.RemoveContainer(dockerClient, containerID); err != nil {
				return errors.Wrap(err, "Failed to activate instance")
			}
		}
		return errors.Wrap(startErr, "Failed to activate instance")
	}

	if err := RecordState(dockerClient, instance, containerID); err != nil {
		return errors.Wrap(err, "Failed to activate instance")
	}
	return nil
}

func IsInstanceActive(instance *model.Instance, host *model.Host) bool {
	if utils.IsNoOp(instance.Data) {
		return true
	}

	client := docker.DefaultClient
	container := utils.GetContainer(client, instance, false)
	return isRunning(client, container)
}

func isRunning(client *client.Client, container *types.Container) bool {
	if container == nil {
		return false
	}
	inspect, err := client.ContainerInspect(context.Background(), container.ID)
	if err == nil {
		return inspect.State.Running
	}
	return false
}

func RecordState(client *client.Client, instance *model.Instance, dockerID string) error {
	if dockerID == "" {
		container := utils.GetContainer(client, instance, false)
		if container != nil {
			dockerID = container.ID
		}
	}

	if dockerID == "" {
		return nil
	}
	contDir := configuration.ContainerStateDir()
	temFilePath := path.Join(contDir, fmt.Sprintf("tmp-%s", dockerID))
	if _, err := os.Stat(temFilePath); err == nil {
		os.Remove(temFilePath)
	}
	filePath := path.Join(contDir, dockerID)
	if _, err := os.Stat(filePath); err == nil {
		os.Remove(filePath)
	}
	if _, err := os.Stat(contDir); err != nil {
		mkErr := os.MkdirAll(contDir, 644)
		if mkErr != nil {
			return mkErr
		}
	}
	data, _ := marshaller.ToString(instance)
	if writeErr := ioutil.WriteFile(temFilePath, data, 0644); writeErr != nil {
		return errors.Wrap(writeErr, "Failed to record state")
	}
	if renameErr := os.Rename(temFilePath, filePath); renameErr != nil {
		return errors.Wrap(renameErr, "Failed to record state")
	}
	return nil
}

func createContainer(client *client.Client, config *container.Config, hostConfig *container.HostConfig,
	imageTag string, instance *model.Instance, name string, progress *progress.Progress) (string, error) {
	logrus.Info("Creating docker container from config")
	labels := config.Labels
	if labels["io.rancher.container.pull_image"] == "always" {
		_, err := DoInstancePull(&model.ImageParams{
			Image:    instance.Image,
			Tag:      "",
			Mode:     "all",
			Complete: false,
		}, progress)
		if err != nil {
			return "", errors.Wrap(err, "Failed to create container")
		}
	}
	config.Image = imageTag

	containerResponse, err := client.ContainerCreate(context.Background(), config, hostConfig, nil, name)
	logrus.Info(fmt.Sprintf("creating container with name %s", name))
	// if image doesn't exist
	if engineCli.IsErrNotFound(err) {
		storage.PullImage(&instance.Image, progress)
		containerResponse, err1 := client.ContainerCreate(context.Background(), config, hostConfig, nil, name)
		if err1 != nil {
			return "", errors.Wrap(err1, "Failed to create container")
		}
		return containerResponse.ID, nil
	}
	return containerResponse.ID, nil
}

func DoInstancePull(params *model.ImageParams, progress *progress.Progress) (types.ImageInspect, error) {
	client := docker.DefaultClient

	imageJSON, ok := utils.GetFieldsIfExist(params.Image.Data, "dockerImage")
	if !ok {
		return types.ImageInspect{}, errors.New("field not exist")
	}
	var dockerImage model.DockerImage
	mapstructure.Decode(imageJSON, &dockerImage)
	existing, _, err := client.ImageInspectWithRaw(context.Background(), dockerImage.FullName, false)
	if err != nil {
		logrus.Error(err)
	}
	if params.Mode == "cached" && err == nil {
		return existing, nil
	}
	if params.Complete {
		_, err1 := client.ImageRemove(context.Background(), dockerImage.FullName+params.Tag, types.ImageRemoveOptions{Force: true})
		if err1 != nil {
			logrus.Error(err1)
		}
		return types.ImageInspect{}, nil
	}
	storage.PullImage(&params.Image, progress)

	if len(params.Tag) > 0 {
		imageInfo := utils.ParseRepoTag(dockerImage.FullName)
		repoTag := fmt.Sprintf("%s:%s", imageInfo["repo"], imageInfo["tag"]+params.Tag)
		logrus.Info(repoTag)
		client.ImageTag(context.Background(), dockerImage.FullName, repoTag)
	}

	inspect, _, err2 := client.ImageInspectWithRaw(context.Background(), dockerImage.FullName, false)
	logrus.Infof("image inspect %v", inspect)
	return inspect, err2
}

func getImageTag(instance *model.Instance) (string, error) {
	var dockerImage model.DockerImage
	mapstructure.Decode(instance.Image.Data["dockerImage"], &dockerImage)
	if dockerImage.FullName == "" {
		return "", errors.New("Can not start container with no image")
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

func setupDNSSearch(hostConfig *container.HostConfig, instance *model.Instance) {
	systemCon := instance.SystemContainer
	if systemCon != "" {
		return
	}
	// if only rancher search is specified,
	// prepend search with params read from the system
	allRancher := true
	dnsSearch := hostConfig.DNSSearch

	if len(dnsSearch) == 0 {
		return
	}
	for _, search := range dnsSearch {
		if strings.HasSuffix(search, "rancher.internal") {
			continue
		}
		allRancher = false
		break
	}

	if !allRancher {
		return
	}

	// read host's resolv.conf
	file, err := os.Open("/etc/resolv.conf")
	if err != nil {
		logrus.Error(err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		s := []string{}
		if strings.HasPrefix(line, "search") {
			// in case multiple search lines
			// respect the last one
			s = strings.Split(line, " ")[1:]
			for i := range s {
				search := s[len(s)-i-1]
				if !utils.SearchInList(s, search) {
					dnsSearch = append(dnsSearch, search)
				}
			}
			hostConfig.DNSSearch = dnsSearch
		}
	}

}

func setupHostname(config *container.Config, instance *model.Instance) {
	config.Hostname = instance.Hostname
}

func setupPorts(config *container.Config, instance *model.Instance,
	hostConfig *container.HostConfig) {
	ports := []model.Port{}
	exposedPorts := map[nat.Port]struct{}{}
	bindings := nat.PortMap{}
	if instance.Ports != nil && len(instance.Ports) > 0 {
		for _, port := range instance.Ports {
			ports = append(ports, model.Port{PrivatePort: port.PrivatePort, Protocol: port.Protocol})
			if port.PrivatePort != 0 {
				bind := nat.Port(fmt.Sprintf("%v/%v", port.PrivatePort, port.Protocol))
				bindAddr := ""
				if bindAddress, ok := utils.GetFieldsIfExist(port.Data, "fields", "bindAddress"); ok {
					bindAddr = utils.InterfaceToString(bindAddress)
				}
				if _, ok := bindings[bind]; !ok {
					bindings[bind] = []nat.PortBinding{nat.PortBinding{HostIP: bindAddr,
						HostPort: utils.ConvertPortToString(port.PublicPort)}}
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

func setupVolumes(config *container.Config, instance *model.Instance,
	hostConfig *container.HostConfig, client *client.Client) {
	if volumes, ok := utils.GetFieldsIfExist(instance.Data, "fields", "dataVolumes"); ok {
		volumes := volumes.([]interface{})
		volumesMap := map[string]struct{}{}
		binds := []string{}
		if len(volumes) > 0 {
			for _, volume := range volumes {
				volume := utils.InterfaceToString(volume)
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
	}

	containers := []string{}
	if vfsList := instance.DataVolumesFromContainers; vfsList != nil {
		for _, vfs := range vfsList {
			var in model.Instance
			mapstructure.Decode(vfs, &in)
			container := utils.GetContainer(client, &in, false)
			if container != nil {
				containers = append(containers, container.ID)
			}
		}
		if containers != nil && len(containers) > 0 {
			hostConfig.VolumesFrom = containers
		}
	}

	if vMounts := instance.VolumesFromDataVolumeMounts; len(vMounts) > 0 {
		for _, vMount := range vMounts {
			storagePool := model.StoragePool{}
			progress := progress.Progress{}
			if !storage.IsVolumeActive(&vMount, &storagePool) {
				storage.DoVolumeActivate(&vMount, &storagePool, &progress)
			}
		}
	}
}

func setupLinks(hostConfig *container.HostConfig, instance *model.Instance) {
	links := []string{}

	if instance.InstanceLinks == nil {
		return
	}
	for _, link := range instance.InstanceLinks {
		if link.TargetInstance.UUID != "" {
			linkStr := fmt.Sprintf("%s:%s", link.TargetInstance.UUID, link.LinkName)
			links = append(links, linkStr)
		}
	}
	hostConfig.Links = links

}

func setupNetworking(instance *model.Instance, host *model.Host,
	config *container.Config, hostConfig *container.HostConfig) {
	client := docker.DefaultClient
	portsSupported, hostnameSupported := setupNetworkMode(instance, client, config, hostConfig)
	setupMacAndIP(instance, config, portsSupported, hostnameSupported)
	setupPortsNetwork(instance, config, hostConfig, portsSupported)
	setupLinksNetwork(instance, config, hostConfig)
	setupIpsec(instance, host, config, hostConfig)
	setupDNS(instance)
}

func flagSystemContainer(instance *model.Instance, config *container.Config) {
	if instance.SystemContainer != "" {
		utils.AddLabel(config, "io.rancher.container.system", instance.SystemContainer)
	}
}

func setupProxy(instance *model.Instance, config *container.Config) {
	if instance.SystemContainer != "" {
		for _, i := range []string{"http_proxy", "https_proxy", "NO_PROXY"} {
			config.Env = append(config.Env, fmt.Sprintf("%v=%v", i, os.Getenv(i)))
		}
	}
}

func setupCattleConfigURL(instance *model.Instance, config *container.Config) {
	if instance.AgentID == 0 && !utils.HasLabel(instance) {
		return
	}

	utils.AddLabel(config, "io.rancher.container.agent_id", strconv.Itoa(instance.AgentID))

	url := configuration.URL()
	logrus.Info("url info " + url)
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

func setupDeviceOptions(hostConfig *container.HostConfig, instance *model.Instance) {

	if deviceOptions, ok := utils.GetFieldsIfExist(instance.Data, "fields", "blkioDeviceOptions"); ok {
		blkioWeightDevice := []*blkiodev.WeightDevice{}
		blkioDeviceReadIOps := []*blkiodev.ThrottleDevice{}
		blkioDeviceWriteBps := []*blkiodev.ThrottleDevice{}
		blkioDeviceReadBps := []*blkiodev.ThrottleDevice{}
		blkioDeviceWriteIOps := []*blkiodev.ThrottleDevice{}

		deviceOptions := deviceOptions.(map[string]interface{})
		for dev, options := range deviceOptions {
			if dev == "DEFAULT_DISK" {
				dev = hostInfo.GetDefaultDisk()
				if dev == "" {
					logrus.Warn(fmt.Sprintf("Couldn't find default device. Not setting device options: %s", options))
					continue
				}
			}
			options := options.(map[string]interface{})
			for key, value := range options {
				value := utils.InterfaceToFloat(value)
				switch key {
				case "weight":
					blkioWeightDevice = append(blkioWeightDevice, &blkiodev.WeightDevice{
						Path:   dev,
						Weight: uint16(value),
					})
					break
				case "readIops":
					blkioDeviceReadIOps = append(blkioDeviceReadIOps, &blkiodev.ThrottleDevice{
						Path: dev,
						Rate: uint64(value),
					})
					break
				case "writeIops":
					blkioDeviceWriteIOps = append(blkioDeviceWriteIOps, &blkiodev.ThrottleDevice{
						Path: dev,
						Rate: uint64(value),
					})
					break
				case "readBps":
					blkioDeviceReadBps = append(blkioDeviceReadBps, &blkiodev.ThrottleDevice{
						Path: dev,
						Rate: uint64(value),
					})
					break
				case "writeBps":
					blkioDeviceWriteBps = append(blkioDeviceWriteBps, &blkiodev.ThrottleDevice{
						Path: dev,
						Rate: uint64(value),
					})
					break
				}
			}
		}
		if len(blkioWeightDevice) > 0 {
			hostConfig.BlkioWeightDevice = blkioWeightDevice
		}
		if len(blkioDeviceReadIOps) > 0 {
			hostConfig.BlkioDeviceReadIOps = blkioDeviceReadIOps
		}
		if len(blkioDeviceWriteIOps) > 0 {
			hostConfig.BlkioDeviceWriteIOps = blkioDeviceWriteIOps
		}
		if len(blkioDeviceReadBps) > 0 {
			hostConfig.BlkioDeviceReadBps = blkioDeviceReadBps
		}
		if len(blkioDeviceWriteBps) > 0 {
			hostConfig.BlkioDeviceWriteBps = blkioDeviceWriteBps
		}
	}
}

func setupLegacyCommand(config *container.Config, fields map[string]interface{}, command string) {
	// This can be removed shortly once cattle removes
	// commandArgs
	logrus.Info("set up command")
	if len(command) == 0 || len(strings.TrimSpace(command)) == 0 {
		return
	}
	commandArgs := []string{}
	if value, ok := utils.GetFieldsIfExist(fields, "commandArgs"); ok {
		for _, v := range value.([]interface{}) {
			commandArgs = append(commandArgs, utils.InterfaceToString(v))
		}
	}
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

func DoInstanceDeactivate(instance *model.Instance, progress *progress.Progress, timeout int) error {
	if utils.IsNoOp(instance.Data) {
		return nil
	}

	client := docker.DefaultClient
	t := time.Duration(timeout) * time.Second
	container := utils.GetContainer(client, instance, false)
	client.ContainerStop(context.Background(), container.ID, &t)
	container = utils.GetContainer(client, instance, false)
	if !isStopped(client, container) {
		client.ContainerKill(context.Background(), container.ID, "KILL")
	}
	if !isStopped(client, container) {
		return fmt.Errorf("Failed to stop container %v", instance.UUID)
	}
	logrus.Infof("container id %v deactivated", container.ID)
	return nil
}

func isStopped(client *client.Client, container *types.Container) bool {
	return !isRunning(client, container)
}

func IsInstanceInactive(instance *model.Instance) bool {
	if utils.IsNoOp(instance.Data) {
		return true
	}

	client := docker.DefaultClient
	container := utils.GetContainer(client, instance, false)
	return isStopped(client, container)
}

func DoInstanceForceStop(request *model.InstanceForceStop) error {
	client := docker.DefaultClient
	time := time.Duration(10)
	if stopErr := client.ContainerStop(context.Background(), request.ID, &time); !engineCli.IsErrNotFound(stopErr) {
		return errors.Wrap(stopErr, "Failed to force stop container")
	} else if stopErr != nil {
		logrus.Infof("container id %v not found", request.ID)
	}
	logrus.Infof("container id %v is forced to be stopped", request.ID)
	return nil
}

func DoInstanceInspect(inspect *model.InstanceInspect) (types.ContainerJSON, error) {
	client := docker.DefaultClient
	containerID := inspect.ID
	if containerID == "" {
		return types.ContainerJSON{}, fmt.Errorf("container with id [%v] not found", containerID)
	}
	containerList, _ := client.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	result := utils.FindFirst(containerList, func(c *types.Container) bool {
		return utils.IDFilter(containerID, c)
	})
	if result == nil {
		name := fmt.Sprintf("/%s", inspect.Name)
		result = utils.FindFirst(containerList, func(c *types.Container) bool {
			return utils.NameFilter(name, c)
		})
	}
	if result != nil {
		logrus.Infof("start inspecting container with id [%s]", result.ID)
		inspectResp, err := client.ContainerInspect(context.Background(), result.ID)
		if err != nil {
			return types.ContainerJSON{}, errors.Wrap(err, "Failed to inspect instance")
		}
		logrus.Infof("container with id [%s] inspected", result.ID)
		return inspectResp, nil
	}
	return types.ContainerJSON{}, fmt.Errorf("container with id [%v] not found", containerID)
}

func IsInstanceRemoved(instance *model.Instance) bool {
	client := docker.DefaultClient
	container := utils.GetContainer(client, instance, false)
	return container == nil
}

func DoInstanceRemove(instance *model.Instance, progress *progress.Progress) error {
	client := docker.DefaultClient
	container := utils.GetContainer(client, instance, false)
	if container == nil {
		return nil
	}
	if err := utils.RemoveContainer(client, container.ID); err != nil {
		return errors.Wrap(err, "fail to remove instance")
	}
	return nil
}

func PurgeState(instance *model.Instance) error {
	client := docker.DefaultClient
	container := utils.GetContainer(client, instance, false)
	if container == nil {
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

func setupResource(fields map[string]interface{}, hostConfig *container.HostConfig) {
	var resource container.Resources
	mapstructure.Decode(fields, &resource)
	hostConfig.Resources = resource
}

// this method convert fields data to fields in host configuration
func setupFieldsHostConfig(fields map[string]interface{}, hostConfig *container.HostConfig) {
	for key, value := range fields {
		switch key {
		case "extraHosts":
			for _, singleValue := range utils.InterfaceToArray(value) {
				if str := utils.InterfaceToString(singleValue); str != "" {
					hostConfig.ExtraHosts = append(hostConfig.ExtraHosts, str)
				}
			}
			break
		case "pidMode":
			hostConfig.PidMode = container.PidMode(utils.InterfaceToString(value))
			break
		case "logConfig":
			vmap := utils.InterfaceToMap(value)
			hostConfig.LogConfig.Type = utils.InterfaceToString(vmap["driver"])
			hostConfig.LogConfig.Config = map[string]string{}
			if vmap["config"] != nil {
				for key1, value1 := range utils.InterfaceToMap(vmap["config"]) {
					hostConfig.LogConfig.Config[key1] = utils.InterfaceToString(value1)
				}
			}
			break
		case "securityOpt":
			for _, singleValue := range utils.InterfaceToArray(value) {
				if str := utils.InterfaceToString(singleValue); str != "" {
					hostConfig.SecurityOpt = append(hostConfig.SecurityOpt, str)
				}
			}
			break
		case "devices":
			hostConfig.Devices = []container.DeviceMapping{}
			for _, singleValue := range utils.InterfaceToArray(value) {
				str := utils.InterfaceToString(singleValue)
				parts := strings.Split(str, ":")
				permission := "rwm"
				if len(parts) == 3 {
					permission = parts[2]
				}
				hostConfig.Devices = append(hostConfig.Devices,
					container.DeviceMapping{
						PathOnHost:        parts[0],
						PathInContainer:   parts[1],
						CgroupPermissions: permission,
					})
			}
			break
		case "dns":
			for _, singleValue := range utils.InterfaceToArray(value) {
				if str := utils.InterfaceToString(singleValue); str != "" {
					hostConfig.DNS = append(hostConfig.DNS, str)
				}
			}
			break
		case "dnsSearch":
			for _, singleValue := range utils.InterfaceToArray(value) {
				if str := utils.InterfaceToString(singleValue); str != "" {
					hostConfig.DNSSearch = append(hostConfig.DNSSearch, str)
				}
			}
			break
		case "capAdd":
			for _, singleValue := range utils.InterfaceToArray(value) {
				if str := utils.InterfaceToString(singleValue); str != "" {
					hostConfig.CapAdd = append(hostConfig.CapAdd, str)
				}
			}
			break
		case "capDrop":
			for _, singleValue := range utils.InterfaceToArray(value) {
				if str := utils.InterfaceToString(singleValue); str != "" {
					hostConfig.CapDrop = append(hostConfig.CapDrop, str)
				}
			}
			break
		case "restartPolicy":
			vmap := utils.InterfaceToMap(value)
			hostConfig.RestartPolicy.Name = utils.InterfaceToString(vmap["name"])
			hostConfig.RestartPolicy.MaximumRetryCount = int(utils.InterfaceToFloat(vmap["maximumRetryCount"]))
			break
		case "cpuShares":
			hostConfig.CPUShares = int64(utils.InterfaceToFloat(value))

		case "volumeDriver":
			hostConfig.VolumeDriver = utils.InterfaceToString(value)
		case "cpuSet":
			hostConfig.CpusetCpus = utils.InterfaceToString(value)
		case "blkioWeight":
			hostConfig.BlkioWeight = uint16(utils.InterfaceToFloat(value))
		case "cgroupParent":
			hostConfig.CgroupParent = utils.InterfaceToString(value)
		case "cpuPeriod":
			hostConfig.CPUPeriod = int64(utils.InterfaceToFloat(value))
		case "cpuQuota":
			hostConfig.CPUQuota = int64(utils.InterfaceToFloat(value))
		case "cpusetMems":
			hostConfig.CpusetMems = utils.InterfaceToString(value)
		case "dnsOpt":
			for _, singleValue := range utils.InterfaceToArray(value) {
				if str := utils.InterfaceToString(singleValue); str != "" {
					hostConfig.CapDrop = append(hostConfig.CapDrop, str)
				}
			}
		case "groupAdd":
			for _, singleValue := range utils.InterfaceToArray(value) {
				if str := utils.InterfaceToString(singleValue); str != "" {
					hostConfig.CapDrop = append(hostConfig.CapDrop, str)
				}
			}
		case "isolation":
			hostConfig.Isolation = container.Isolation(utils.InterfaceToString(value))
		case "kernelMemory":
			hostConfig.KernelMemory = int64(utils.InterfaceToFloat(value))
		case "memoryReservation":
			hostConfig.MemoryReservation = int64(utils.InterfaceToFloat(value))
		case "memorySwap":
			hostConfig.MemorySwap = int64(utils.InterfaceToFloat(value))
		case "MemorySwappiness":
			ms := int64(utils.InterfaceToFloat(value))
			hostConfig.MemorySwappiness = &ms
		case "oomKillDisable":
			od := utils.InterfaceToBool(value)
			hostConfig.OomKillDisable = &od
		case "shmSize":
			hostConfig.ShmSize = int64(utils.InterfaceToFloat(value))
		case "tmpfs":
			vmap := utils.InterfaceToMap(value)
			hostConfig.Tmpfs = map[string]string{}
			for k, v := range vmap {
				hostConfig.Tmpfs[k] = utils.InterfaceToString(v)
			}
		case "ulimits":
			vmap := utils.InterfaceToMap(value)
			ul := units.Ulimit{
				Name: utils.InterfaceToString(vmap["name"]),
				Hard: int64(utils.InterfaceToFloat(vmap["hard"])),
				Soft: int64(utils.InterfaceToFloat(vmap["soft"])),
			}
			hostConfig.Ulimits = []*units.Ulimit{}
			hostConfig.Ulimits = append(hostConfig.Ulimits, &ul)
		case "uts":
			hostConfig.UTSMode = container.UTSMode(utils.InterfaceToString(value))
		}
	}
}

// this method convert fields data to fields in configuration
func setupFieldsConfig(fields map[string]interface{}, config *container.Config) {
	for key, value := range fields {
		switch key {
		case "command":
			commands, ok := value.([]interface{})
			if !ok {
				setupLegacyCommand(config, fields, utils.InterfaceToString(value))
			} else {
				cmds := utils.InterfaceToArray(commands)
				for _, cmd := range cmds {
					config.Cmd = append(config.Cmd, utils.InterfaceToString(cmd))
				}
			}
		case "environment":
			for k, v := range utils.InterfaceToMap(value) {
				config.Env = append(config.Env, fmt.Sprintf("%v=%v", k, utils.InterfaceToString(v)))
			}
		case "workingDir":
			config.WorkingDir = utils.InterfaceToString(value)
			break
		case "entryPoint":
			for _, singleValue := range utils.InterfaceToArray(value) {
				if str := utils.InterfaceToString(singleValue); str != "" {
					config.Entrypoint = append(config.Entrypoint, str)
				}
			}
			break
		case "tty":
			config.Tty = utils.InterfaceToBool(value)
			break
		case "stdinOpen":
			config.OpenStdin = utils.InterfaceToBool(value)
			break
		case "domainName":
			config.Domainname = utils.InterfaceToString(value)
			break
		case "labels":
			for k, v := range utils.InterfaceToMap(value) {
				str := utils.InterfaceToString(v)
				config.Labels[k] = str
			}
		case "stopSignal":
			config.StopSignal = utils.InterfaceToString(value)
		}

	}
}

func setupNetworkingConfig(networkConfig *network.NetworkingConfig, instance *model.Instance) {

}

func setupLabels(labels map[string]interface{}, config *container.Config) {
	for k, v := range labels {
		config.Labels[k] = utils.InterfaceToString(v)
	}
}

func setupPublishPorts(hostConfig *container.HostConfig, instance *model.Instance) {
	portsPub, ok := utils.GetFieldsIfExist(instance.Data, "fields", "publishAllPorts")
	if ok {
		hostConfig.PublishAllPorts = utils.InterfaceToBool(portsPub)
	}
}

func setupMacAndIP(instance *model.Instance, config *container.Config, setMac bool, setHostname bool) {
	/*
		Configures the mac address and primary ip address for the the supplied
		container. The macAddress is configured directly as part of the native
		docker API. The primary IP address is set as an environment variable on the
		container. Another Rancher micro-service will detect this environment
		variable when the container is started and inject the IP into the
		container.

		Note: while an instance can technically have more than one nic based on the
		resource schema, this implementation assumes a single nic for the purpose
		of configuring the mac address and IP.
	*/
	macAddress := ""
	deviceNumber := -1
	for _, nic := range instance.Nics {
		if deviceNumber == -1 {
			macAddress = nic.MacAddress
			deviceNumber = nic.DeviceNumber
		} else if deviceNumber > nic.DeviceNumber {
			macAddress = nic.MacAddress
			deviceNumber = nic.DeviceNumber
		}
	}
	if setMac {
		config.MacAddress = macAddress
	}

	if !setHostname {
		config.Hostname = ""
	}

	if instance.Nics != nil && len(instance.Nics) > 0 && instance.Nics[0].IPAddresses != nil {
		// Assume one nic
		nic := instance.Nics[0]
		ipAddress := ""
		for _, ip := range nic.IPAddresses {
			if ip.Role == "primary" {
				ipAddress = fmt.Sprintf("%s/%s", ip.Address, strconv.Itoa(ip.Subnet.CidrSize))
				break
			}
		}
		if ipAddress != "" {
			utils.AddLabel(config, "io.rancher.container.ip", ipAddress)
		}
	}
}

func setupNetworkMode(instance *model.Instance, client *client.Client,
	config *container.Config, hostConfig *container.HostConfig) (bool, bool) {
	/*
			Based on the network configuration we choose the network mode to set in
		    Docker.  We only really look for none, host, or container.  For all
		    all other configurations we assume bridge mode
	*/
	portsSupported := true
	hostnameSupported := true
	if len(instance.Nics) > 0 {
		kind := instance.Nics[0].Network.Kind
		if kind == "dockerHost" {
			portsSupported = false
			hostnameSupported = false
			config.NetworkDisabled = false
			hostConfig.NetworkMode = "host"
			hostConfig.Links = nil
		} else if kind == "dockerNone" {
			portsSupported = false
			config.NetworkDisabled = true
			hostConfig.NetworkMode = "none"
			hostConfig.Links = nil
		} else if kind == "dockerContainer" {
			portsSupported = false
			hostnameSupported = false
			id := instance.NetworkContainer["uuid"]
			var in model.Instance
			mapstructure.Decode(instance.NetworkContainer, &in)
			other := utils.GetContainer(client, &in, false)
			if other != nil {
				id = other.ID
			}
			hostConfig.NetworkMode = container.NetworkMode(fmt.Sprintf("container:%v", id))
			hostConfig.Links = nil
		}
	}
	return portsSupported, hostnameSupported

}

func setupPortsNetwork(instance *model.Instance, config *container.Config,
	hostConfig *container.HostConfig, portsSupported bool) {
	/*
			Docker 1.9+ does not allow you to pass port info for networks that don't
		    support ports (net, none, container:x)
	*/
	if !portsSupported {
		hostConfig.PublishAllPorts = false
		config.ExposedPorts = map[nat.Port]struct{}{}
		hostConfig.PortBindings = nat.PortMap{}
	}
}

func setupIpsec(instance *model.Instance, host *model.Host, config *container.Config,
	hostConfig *container.HostConfig) {
	/*
			If the supplied instance is a network agent, configures the ports needed
		    to achieve multi-host networking.
	*/
	networkAgent := false
	if instance.SystemContainer == "" || instance.SystemContainer == "NetworkAgent" {
		networkAgent = true
	}
	if !networkAgent || !utils.HasService(instance, "ipsecTunnelService") {
		return
	}
	hostID := strconv.Itoa(host.ID)
	if data, ok := utils.GetFieldsIfExist(instance.Data, "ipsec", hostID); ok {
		info := utils.InterfaceToMap(data)
		natValue := utils.InterfaceToFloat(info["nat"])
		isakmp := utils.InterfaceToFloat(info["isakmp"])

		binding := hostConfig.PortBindings

		port1 := nat.Port(fmt.Sprintf("%v/%v", 500, "udp"))
		port2 := nat.Port(fmt.Sprintf("%v/%v", 4500, "udp"))
		bind1 := nat.PortBinding{HostIP: "0.0.0.0", HostPort: strconv.Itoa(int(isakmp))}
		bind2 := nat.PortBinding{HostIP: "0.0.0.0", HostPort: strconv.Itoa(int(natValue))}
		exposedPorts := map[nat.Port]struct{}{
			port1: struct{}{},
			port2: struct{}{},
		}
		if _, ok := binding[port1]; ok {
			binding[port1] = append(binding[port1], bind1)
		} else {
			binding[port1] = []nat.PortBinding{bind1}
		}
		if _, ok := binding[port2]; ok {
			binding[port2] = append(binding[port2], bind1)
		} else {
			binding[port2] = []nat.PortBinding{bind2}
		}
		if len(config.ExposedPorts) > 0 {
			existingMap := config.ExposedPorts
			for port := range exposedPorts {
				existingMap[port] = struct{}{}
			}
			config.ExposedPorts = existingMap
		} else {
			config.ExposedPorts = exposedPorts
		}

	}
}

func setupDNS(instance *model.Instance) {
	if !utils.HasService(instance, "dnsService") || instance.Kind == "virtualMachine" {
		return
	}
	ipAddress, macAddress, subnet := utils.FindIPAndMac(instance)

	if ipAddress == "" || macAddress == "" {
		return
	}

	parts := strings.Split(ipAddress, ".")
	if len(parts) != 4 {
		return
	}
	part2, _ := strconv.Atoi(parts[2])
	part3, _ := strconv.Atoi(parts[3])
	mark := strconv.Itoa(part2*1000 + part3)

	utils.CheckOutput([]string{"iptables", "-w", "-t", "nat", "-A", "CATTLE_PREROUTING",
		"!", "-s", subnet, "-d", "169.254.169.250", "-m", "mac",
		"--mac-source", macAddress, "-j", "MARK", "--set-mark",
		mark})
	utils.CheckOutput([]string{"iptables", "-w", "-t", "nat", "-A", "CATTLE_POSTROUTING",
		"!", "-s", subnet, "-d", "169.254.169.250", "-m", "mark", "--mark", mark,
		"-j", "SNAT", "--to", ipAddress})

}

func setupLinksNetwork(instance *model.Instance, config *container.Config,
	hostConfig *container.HostConfig) {
	/*
			Sets up a container's config for rancher-managed links by removing the
		    docker native link configuration and emulating links through environment
		    variables.

		    Note that a non-rancher container (one created and started outside the
		    rancher API) container will not have its link configuration manipulated.
		    This is because on a container restart, we would not be able to properly
		    rebuild the link configuration because it depends on manipulating the
		    createConfig.
	*/
	if !utils.HasService(instance, "linkService") || utils.IsNonrancherContainer(instance) {
		return
	}

	hostConfig.Links = nil

	result := map[string]string{}
	if instance.InstanceLinks != nil {
		for _, link := range instance.InstanceLinks {
			linkName := link.LinkName
			utils.AddLinkEnv(linkName, link, result, "")
			utils.CopyLinkEnv(linkName, link, result)
			if names, ok := utils.GetFieldsIfExist(link.Data, "fields", "instanceNames"); ok {
				for _, name := range names.([]interface{}) {
					name := utils.InterfaceToString(name)
					utils.AddLinkEnv(name, link, result, linkName)
					utils.CopyLinkEnv(name, link, result)
					// This does assume the format {env}_{name}
					parts := strings.SplitN(name, "_", 2)
					if len(parts) == 1 {
						continue
					}
					utils.AddLinkEnv(parts[1], link, result, linkName)
					utils.CopyLinkEnv(parts[1], link, result)
				}

			}
		}
		if len(result) > 0 {
			utils.AddToEnv(config, result)
		}
	}

}
