package compute

import (
	"fmt"

	urls "net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/pkg/errors"
	"github.com/rancher/agent/core/progress"
	"github.com/rancher/agent/core/storage"
	"github.com/rancher/agent/model"
	configuration "github.com/rancher/agent/utilities/config"
	"github.com/rancher/agent/utilities/constants"
	"github.com/rancher/agent/utilities/utils"
	"golang.org/x/net/context"
)

var (
	dockerRootOnce = sync.Once{}
	dockerRoot     = ""
)

func createContainer(dockerClient *client.Client, config *container.Config, hostConfig *container.HostConfig, networkConfig *network.NetworkingConfig,
	imageTag string, instance model.Instance, name string, progress *progress.Progress) (string, error) {
	labels := config.Labels
	if labels[constants.PullImageLabels] == "always" {
		params := model.ImageParams{
			Tag:       "",
			Mode:      "all",
			Complete:  false,
			ImageUUID: instance.Data.Fields.ImageUUID,
		}
		_, err := DoInstancePull(params, progress, dockerClient, instance.Data.Fields.Build, instance.ImageCredential)
		if err != nil {
			return "", errors.Wrap(err, constants.CreateContainerError+"failed to pull instance")
		}
	}
	imageName := utils.ParseRepoTag(imageTag)
	config.Image = imageName

	containerResponse, err := dockerContainerCreate(context.Background(), dockerClient, config, hostConfig, networkConfig, name)
	// if image doesn't exist
	if client.IsErrImageNotFound(err) {
		if err := storage.PullImage(progress, dockerClient, imageTag, instance.Data.Fields.Build, instance.ImageCredential); err != nil {
			return "", errors.Wrap(err, constants.CreateContainerError+"failed to pull image")
		}
		containerResponse, err1 := dockerContainerCreate(context.Background(), dockerClient, config, hostConfig, networkConfig, name)
		if err1 != nil {
			return "", errors.Wrap(err1, constants.CreateContainerError+"failed to create container")
		}
		return containerResponse.ID, nil
	} else if err != nil {
		return "", errors.Wrap(err, constants.CreateContainerError+"failed to create container")
	}
	return containerResponse.ID, nil
}

func getImageTag(instance model.Instance) (string, error) {
	dockerImage := instance.Data.Fields.ImageUUID
	if dockerImage == "" {
		return "", errors.New(constants.StartContainerNoImageError + "the full name of docker image is empty")
	}
	return dockerImage, nil
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

func setupPorts(config *container.Config, instance model.Instance, hostConfig *container.HostConfig) {
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
						{
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

func getDockerRoot(client *client.Client) string {
	dockerRootOnce.Do(func() {
		info, err := client.Info(context.Background())
		if err != nil {
			panic(err.Error())
		}
		dockerRoot = info.DockerRootDir
	})
	return dockerRoot
}

func setupVolumes(config *container.Config, instance model.Instance, hostConfig *container.HostConfig, client *client.Client, progress *progress.Progress) error {

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

				// Redirect /var/lib/docker:/var/lib/docker to where Docker root really is
				if parts[0] == "/var/lib/docker" && parts[1] == "/var/lib/docker" {
					root := getDockerRoot(client)
					if root != "/var/lib/docker" {
						volumesMap[root] = struct{}{}
						binds = append(binds, fmt.Sprintf("%s:%s:%s", root, parts[1], mode))
						binds = append(binds, fmt.Sprintf("%s:%s:%s", root, root, mode))
						continue
					}
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
					return errors.Wrap(err, constants.SetupVolumesError+"failed to get container")
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
			// volume active == exists, possibly not attached to this host
			if ok, err := storage.IsVolumeActive(vMount, storagePool, client); !ok && err == nil {
				if err := storage.DoVolumeActivate(vMount, storagePool, progress, client); err != nil {
					return errors.Wrap(err, constants.SetupVolumesError+"failed to activate volume")
				}
			} else if err != nil {
				return errors.Wrap(err, constants.SetupVolumesError+"failed to check whether volume is activated")
			}
			if storage.IsRancherVolume(vMount) {
				progress.Update(fmt.Sprintf("Attaching volume %s", vMount.Name), "yes", nil)
				if err := storage.RancherStorageVolumeAttach(vMount); err != nil {
					return errors.Wrap(err, constants.SetupVolumesError+"failed to attach volume")
				}
			}
		}
	}
	return nil
}

func setupHeathConfig(instanceFields model.InstanceFields, config *container.Config) {
	healthConfig := &container.HealthConfig{}
	healthConfig.Test = instanceFields.HealthCmd
	healthConfig.Interval = time.Duration(instanceFields.HealthInterval) * time.Second
	healthConfig.Retries = instanceFields.HealthRetries
	healthConfig.Timeout = time.Duration(instanceFields.HealthTimeout) * time.Second
	config.Healthcheck = healthConfig
}

func setupProxy(instance model.Instance, config *container.Config, hostEntries map[string]string) {
	// only setup envs for system container
	if instance.System {
		envMap := map[string]string{}
		envList := []string{}
		for _, env := range config.Env {
			/*
				3 case:
				1. foo=bar. Parse as normal
				2. foo=. Parse as foo=
				3. foo. Parse as foo
			*/
			part := strings.SplitN(env, "=", 2)
			if len(part) == 1 {

				if strings.Contains(env, "=") {
					//case 2
					envMap[part[0]] = ""
				} else {
					envList = append(envList, env)
				}
			} else if len(part) == 2 {
				envMap[part[0]] = part[1]
			}
		}
		for _, key := range constants.HTTPProxyList {
			if hostEntries[key] != "" {
				envMap[key] = hostEntries[key]
			}
		}
		envs := []string{}
		for _, env := range envList {
			if _, ok := envMap[env]; ok {
				continue
			}
			envs = append(envs, env)
		}
		for key, value := range envMap {
			envs = append(envs, fmt.Sprintf("%v=%v", key, value))
		}
		config.Env = envs
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

func setupNetworkingConfig(networkConfig *network.NetworkingConfig, instance model.Instance) {
}

func setupLabels(labels map[string]string, config *container.Config) {
	for k, v := range labels {
		config.Labels[k] = utils.InterfaceToString(v)
	}
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

	config.StopSignal = fields.StopSignal

	config.User = fields.User
}

func isStopped(client *client.Client, container types.Container) (bool, error) {
	ok, err := isRunning(client, container)
	if err != nil {
		return false, err
	}
	return !ok, nil
}

func isRunning(dockerClient *client.Client, container types.Container) (bool, error) {
	if container.ID == "" {
		return false, nil
	}
	inspect, err := dockerClient.ContainerInspect(context.Background(), container.ID)
	if err == nil {
		return inspect.State.Running, nil
	} else if client.IsErrContainerNotFound(err) {
		return false, nil
	}
	return false, err
}

func getHostEntries() map[string]string {
	data := map[string]string{}
	for _, env := range constants.HTTPProxyList {
		if os.Getenv(env) != "" {
			data[env] = os.Getenv(env)
		}
	}
	return data
}
