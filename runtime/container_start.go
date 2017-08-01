package runtime

import (
	"context"
	"fmt"
	urls "net/url"
	"strconv"
	"strings"
	"time"

	"os"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	v2 "github.com/rancher/go-rancher/v2"
	"github.com/pkg/errors"
	"github.com/rancher/agent/progress"
	"github.com/rancher/agent/utils"
	"sync"
)

const (
	ContainerNameLabel = "io.rancher.container.name"
	PullImageLabels    = "io.rancher.container.pull_image"
	UUIDLabel          = "io.rancher.container.uuid"
	AgentIDLabel       = "io.rancher.container.agent_id"
)

var (
	dockerRootOnce = sync.Once{}
	dockerRoot     = ""
	HTTPProxyList = []string{"http_proxy", "HTTP_PROXY", "https_proxy", "HTTPS_PROXY", "no_proxy", "NO_PROXY"}
)

func ContainerStart(containerSpec v2.Container, volumes []v2.Volume, networks []v2.Network, credentials []v2.Credential, progress *progress.Progress, runtimeClient *client.Client, idsMap map[string]string) error {
	started := false

	// setup name
	parts := strings.Split(containerSpec.Uuid, "-")
	if len(parts) == 0 {
		return errors.New("Failed to parse UUID")
	}
	name := fmt.Sprintf("r-%s", containerSpec.Uuid)
	if str := utils.NameRegexCompiler.FindString(containerSpec.Name); str != "" {
		name = fmt.Sprintf("r-%s-%s", containerSpec.Name, parts[0])
	}

	// creating managed volumes
	rancherBindMounts, err := setupRancherFlexVolume(volumes, containerSpec.DataVolumes, progress)
	if err != nil {
		return errors.Wrap(err, "failed to set up rancher flex volumes")
	}

	// make sure managed volumes are unmounted if container is not started
	defer func() {
		if !started {
			unmountRancherFlexVolume(volumes)
		}
	}()

	// setup container spec(config and hostConfig)
	spec, err := setupContainerSpec(containerSpec, volumes, networks, rancherBindMounts, runtimeClient, progress, idsMap)
	if err != nil {
		return errors.Wrap(err, "failed to generate container spec")
	}

	containerId, err := utils.FindContainer(runtimeClient, containerSpec, false)
	if err != nil {
		if !utils.IsContainerNotFoundError(err) {
			return errors.Wrap(err, "failed to get container")
		}
	}
	created := false
	if containerId == "" {
		credential := v2.Credential{}
		if credentials != nil && len(credentials) > 0 {
			credential = credentials[0]
		}
		newID, err := createContainer(runtimeClient, &spec.config, &spec.hostConfig, containerSpec, credential, name, progress)
		if err != nil {
			return errors.Wrap(err, "failed to create container")
		}
		containerId = newID
		created = true
	}

	startErr := utils.Serialize(func() error {
		return runtimeClient.ContainerStart(context.Background(), containerId, types.ContainerStartOptions{})
	})
	if startErr != nil {
		if created {
			if err := utils.RemoveContainer(runtimeClient, containerId); err != nil {
				return errors.Wrap(err, "failed to remove container")
			}
		}
		return errors.Wrap(startErr, "failed to start container")
	}

	logrus.Infof("rancher id [%v]: Container [%v] with docker id [%v] has been started", containerSpec.Id, containerSpec.Name, containerId)
	started = true
	return nil
}

func IsContainerStarted(containerSpec v2.Container, client *client.Client) (bool, error) {
	cont, err := utils.FindContainer(client, containerSpec, false)
	if err != nil {
		if utils.IsContainerNotFoundError(err) {
			return false, nil
		}
		return false, errors.Wrap(err, "failed to get container")
	}
	return isRunning(client, cont)
}

type dockerContainerSpec struct {
	config     container.Config
	hostConfig container.HostConfig
}

func setupContainerSpec(containerSpec v2.Container, volumes []v2.Volume, networks []v2.Network, rancherBindMounts []string, runtimeClient *client.Client, progress *progress.Progress, idsMap map[string]string) (dockerContainerSpec, error) {
	config := container.Config{
		OpenStdin: true,
	}
	hostConfig := container.HostConfig{
		PublishAllPorts: false,
		Privileged:      containerSpec.Privileged,
		ReadonlyRootfs:  containerSpec.ReadOnly,
	}

	initializeMaps(&config, &hostConfig)

	setupLabels(containerSpec.Labels, &config)

	config.Labels[UUIDLabel] = containerSpec.Uuid
	config.Labels[ContainerNameLabel] = containerSpec.Name

	setupFieldsHostConfig(containerSpec, &hostConfig)

	setupFieldsConfig(containerSpec, &config)

	setupPublishPorts(&hostConfig, containerSpec)

	if err := setupDNSSearch(&hostConfig, containerSpec); err != nil {
		return dockerContainerSpec{}, errors.Wrap(err, "failed to set up DNS search")
	}

	setupHostname(&config, containerSpec)

	setupPorts(&config, containerSpec, &hostConfig)

	hostConfig.Binds = append(hostConfig.Binds, rancherBindMounts...)

	if err := setupNonRancherVolumes(&config, volumes, containerSpec, &hostConfig, runtimeClient, progress, idsMap); err != nil {
		return dockerContainerSpec{}, errors.Wrap(err, "failed to set up volumes")
	}

	if err := setupNetworking(containerSpec, &config, &hostConfig, idsMap, networks); err != nil {
		return dockerContainerSpec{}, errors.Wrap(err, "failed to set up networking")
	}

	setupProxy(containerSpec, &config, getHostEntries())

	setupCattleConfigURL(containerSpec, &config)

	setupDeviceOptions(&hostConfig, containerSpec)

	setupComputeResourceFields(&hostConfig, containerSpec)

	setupHeathConfig(containerSpec, &config)
	return dockerContainerSpec{
		config:     config,
		hostConfig: hostConfig,
	}, nil
}

type PullParams struct {
	Tag       string
	Mode      string
	Complete  bool
	ImageUUID string
}

func createContainer(dockerClient *client.Client, config *container.Config, hostConfig *container.HostConfig, containerSpec v2.Container, credential v2.Credential, name string, progress *progress.Progress) (string, error) {
	labels := config.Labels
	if labels[PullImageLabels] == "always" {
		params := PullParams{
			Tag:       "",
			Mode:      "all",
			Complete:  false,
			ImageUUID: containerSpec.Image,
		}
		_, err := DoInstancePull(params, progress, dockerClient, credential)
		if err != nil {
			return "", errors.Wrap(err, "failed to pull instance")
		}
	}
	config.Image = containerSpec.Image

	containerResponse, err := dockerContainerCreate(context.Background(), dockerClient, config, hostConfig, name)
	// if image doesn't exist
	if client.IsErrImageNotFound(err) {
		if err := ImagePull(progress, dockerClient, containerSpec.Image, credential); err != nil {
			return "", errors.Wrap(err, "failed to pull image")
		}
		containerResponse, err1 := dockerContainerCreate(context.Background(), dockerClient, config, hostConfig, name)
		if err1 != nil {
			return "", errors.Wrap(err1, "failed to create container")
		}
		return containerResponse.ID, nil
	} else if err != nil {
		return "", errors.Wrap(err, "failed to create container")
	}
	return containerResponse.ID, nil
}

func getImageTag(containerSpec v2.Container) (string, error) {
	dockerImage := containerSpec.ImageUuid
	if dockerImage == "" {
		return "", errors.New("the full name of docker image is empty")
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

func setupHostname(config *container.Config, containerSpec v2.Container) {
	config.Hostname = containerSpec.Hostname
}

func setupPorts(config *container.Config, containerSpec v2.Container, hostConfig *container.HostConfig) {
	//ports := []types.Port{}
	exposedPorts := map[nat.Port]struct{}{}
	bindings := nat.PortMap{}
	for _, endpoint := range containerSpec.PublicEndpoints {
		if endpoint.PrivatePort != 0 {
			bind := nat.Port(fmt.Sprintf("%v/%v", endpoint.PrivatePort, endpoint.Protocol))
			bindAddr := endpoint.BindIpAddress
			if _, ok := bindings[bind]; !ok {
				bindings[bind] = []nat.PortBinding{
					{
						HostIP:   bindAddr,
						HostPort: strconv.Itoa(int(endpoint.PublicPort)),
					},
				}
			} else {
				bindings[bind] = append(bindings[bind], nat.PortBinding{
					HostIP: bindAddr,
					HostPort: strconv.Itoa(int(endpoint.PublicPort)),
				})
			}
			exposedPorts[bind] = struct{}{}
		}

	}
	fmt.Printf("#### %+v", bindings)

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

// setupVolumes volumes except rancher specific volumes. For rancher-managed volume driver they will be setup through special steps like flexvolume
func setupNonRancherVolumes(config *container.Config, volumes []v2.Volume, containerSpec v2.Container, hostConfig *container.HostConfig, client *client.Client, progress *progress.Progress, idsMap map[string]string) error {
	volumesMap := map[string]struct{}{}
	binds := []string{}

	rancherManagedVolumeNames := map[string]struct{}{}
	for _, volume := range volumes {
		if IsRancherVolume(volume) {
			rancherManagedVolumeNames[volume.Name] = struct{}{}
		}
	}


	for _, volume := range containerSpec.DataVolumes {
		parts := strings.SplitN(volume, ":", 3)
		// don't set rancher managed volume
		if _, ok := rancherManagedVolumeNames[parts[0]]; ok {
			continue
		}
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
		config.Volumes = volumesMap
		hostConfig.Binds = append(hostConfig.Binds, binds...)
	}


	containers := []string{}
	if containerSpec.DataVolumesFrom != nil {
		for _, volumeFrom := range containerSpec.DataVolumesFrom {
			if idsMap[volumeFrom] != "" {
				containers = append(containers, idsMap[volumeFrom])
			}
		}
		if len(containers) > 0 {
			hostConfig.VolumesFrom = containers
		}
	}

	for _, volume := range volumes {
		// volume active == exists, possibly not attached to this host
		if !IsRancherVolume(volume) {
			if ok, err := IsVolumeActive(volume, client); !ok && err == nil {
				if err := DoVolumeActivate(volume, client, progress); err != nil {
					return errors.Wrap(err, "failed to activate volume")
				}
			} else if err != nil {
				return errors.Wrap(err, "failed to check whether volume is activated")
			}
		}
	}

	return nil
}

func setupHeathConfig(spec v2.Container, config *container.Config) {
	healthConfig := &container.HealthConfig{}
	healthConfig.Test = spec.HealthCmd
	healthConfig.Interval = time.Duration(spec.HealthInterval) * time.Second
	healthConfig.Retries = int(spec.HealthRetries)
	healthConfig.Timeout = time.Duration(spec.HealthTimeout) * time.Second
	config.Healthcheck = healthConfig
}

func setupProxy(containerSpec v2.Container, config *container.Config, hostEntries map[string]string) {
	// only setup envs for system container
	if containerSpec.System {
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
		for _, key := range HTTPProxyList {
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

func setupCattleConfigURL(containerSpec v2.Container, config *container.Config) {
	if containerSpec.AgentId == "" && !utils.HasLabel(containerSpec) {
		return
	}

	utils.AddLabel(config, AgentIDLabel, containerSpec.AgentId)

	url := utils.URL()
	if len(url) > 0 {
		parsed, err := urls.Parse(url)
		if err != nil {
			logrus.Error(err)
		} else {
			if strings.Contains(parsed.Host, "localhost") {
				port := utils.APIProxyListenPort()
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

func setupLabels(labels map[string]interface{}, config *container.Config) {
	for k, v := range labels {
		config.Labels[k] = utils.InterfaceToString(v)
	}
}

// this method convert fields data to fields in configuration
func setupFieldsConfig(spec v2.Container, config *container.Config) {

	config.Cmd = spec.Command

	envs := []string{}
	for k, v := range spec.Environment {
		envs = append(envs, fmt.Sprintf("%v=%v", k, v))
	}
	config.Env = append(config.Env, envs...)

	config.WorkingDir = spec.WorkingDir

	config.Entrypoint = spec.EntryPoint

	config.Tty = spec.Tty

	config.OpenStdin = spec.StdinOpen

	config.Domainname = spec.DomainName

	config.StopSignal = spec.StopSignal

	config.User = spec.User
}

func isRunning(dockerClient *client.Client, containerId string) (bool, error) {
	if containerId == "" {
		return false, nil
	}
	inspect, err := dockerClient.ContainerInspect(context.Background(), containerId)
	if err == nil {
		return inspect.State.Running, nil
	} else if client.IsErrContainerNotFound(err) {
		return false, nil
	}
	return false, err
}

func getHostEntries() map[string]string {
	data := map[string]string{}
	for _, env := range HTTPProxyList {
		if os.Getenv(env) != "" {
			data[env] = os.Getenv(env)
		}
	}
	return data
}
