package utils

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/blkiodev"
	"github.com/docker/engine-api/types/container"
	"github.com/docker/engine-api/types/filters"
	"github.com/docker/engine-api/types/strslice"
	"github.com/docker/go-connections/nat"
	"github.com/mitchellh/mapstructure"
	"github.com/rancher/agent/handlers/docker"
	"github.com/rancher/agent/handlers/marshaller"
	"github.com/rancher/agent/handlers/progress"
	"github.com/rancher/agent/model"
	"github.com/rancher/go-machine-service/events"
	revents "github.com/rancher/go-machine-service/events"
	"golang.org/x/net/context"
	"io/ioutil"
	urls "net/url"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var CreateConfigFields = []model.Tuple{
	model.Tuple{Src: "labels", Dest: "labels"},
	model.Tuple{Src: "environment", Dest: "environment"},
	model.Tuple{Src: "directory'", Dest: "workingDir"},
	model.Tuple{Src: "domainName", Dest: "domainname"},
	model.Tuple{Src: "memory", Dest: "mem_limit"},
	model.Tuple{Src: "memorySwap", Dest: "memswap_limit"},
	model.Tuple{Src: "cpuSet", Dest: "cpuset"},
	model.Tuple{Src: "cpuShares", Dest: "cpu_shares"},
	model.Tuple{Src: "tty", Dest: "tty"},
	model.Tuple{Src: "stdinOpen", Dest: "stdin_open"},
	model.Tuple{Src: "detach", Dest: "detach"},
	model.Tuple{Src: "workingDir", Dest: "working_dir"},
	model.Tuple{Src: "labels", Dest: "labels"},
	model.Tuple{Src: "entryPoint", Dest: "entrypoint"},
}

var StartConfigFields = []model.Tuple{
	model.Tuple{Src: "capAdd", Dest: "cap_add"},
	model.Tuple{Src: "capDrop", Dest: "cap_drop"},
	model.Tuple{Src: "dnsSearch", Dest: "dnsSearch"},
	model.Tuple{Src: "dns", Dest: "dns"},
	model.Tuple{Src: "extraHosts", Dest: "extra_hosts"},
	model.Tuple{Src: "publishAllPorts", Dest: "publish_all_ports"},
	model.Tuple{Src: "lxcConf", Dest: "lxc_conf"},
	model.Tuple{Src: "logConfig", Dest: "logConfig"},
	model.Tuple{Src: "securityOpt", Dest: "security_opt"},
	model.Tuple{Src: "restartPolicy", Dest: "restart_policy"},
	model.Tuple{Src: "pidMode", Dest: "pid_mode"},
	model.Tuple{Src: "devices", Dest: "devices"},
}

func DoInstanceActivate(instance *model.Instance, host *model.Host, progress *progress.Progress) error {
	if isNoOp(instance.Data) {
		return nil
	}
	client := docker.GetClient(DefaultVersion)

	imageTag, err := getImageTag(instance)
	if err != nil {
		logrus.Debug(err)
		return err
	}
	name := instance.UUID
	instanceName := instance.Name
	if len(instanceName) > 0 {
		if ok, _ := regexp.Match("^[a-zA-Z0-9][a-zA-Z0-9_.-]+$", []byte(instanceName)); ok {
			id := fmt.Sprintf("r-%s", instanceName)
			_, err1 := client.ContainerInspect(context.Background(), id)
			if err1 != nil {
				name = id
			}
		}
	}
	logrus.Info(name)
	var createConfig = map[string]interface{}{
		"name":   name,
		"detach": true,
	}

	var startConfig = map[string]interface{}{
		"publishAllPorts": false,
		"privileged":      isTrue(instance, "privileged"),
		"ReadonlyRootfs":  isTrue(instance, "readOnly"),
	}

	// These _setupSimpleConfigFields calls should happen before all
	// other config because they stomp over config fields that other
	// setup methods might append to. Example: the environment field
	// setupSimpleConfigFields(createConfig, instance, CreateConfigFields)

	// setupSimpleConfigFields(startConfig, instance, StartConfigFields)

	addLabel(createConfig, map[string]string{UUIDLabel: instance.UUID})

	if len(instanceName) > 0 {
		addLabel(createConfig, map[string]string{"io.rancher.container.name": instanceName})
	}

	setupDNSSearch(startConfig, instance)

	setupLogging(startConfig, instance)

	setupHostname(createConfig, instance)

	setupCommand(createConfig, instance)

	setupPorts(createConfig, instance, startConfig)

	setupVolumes(createConfig, instance, startConfig, client)

	setupLinks(startConfig, instance)

	setupNetworking(instance, host, createConfig, startConfig)

	flagSystemContainer(instance, createConfig)

	setupProxy(instance, createConfig)

	setupCattleConfigURL(instance, createConfig)

	hostConfig := createHostConfig(startConfig)
	setupResource(instance.Data["fields"].(map[string]interface{}), &hostConfig)
	setupDeviceOptions(&hostConfig, instance)

	var config container.Config
	mapstructure.Decode(createConfig, &config)
	setupConfig(instance.Data["fields"].(map[string]interface{}), &config)

	//debug
	logrus.Infof("container configuration %+v\n", config)
	logrus.Infof("container host configuration %+v\n", hostConfig)

	container := GetContainer(client, instance, false)
	containerID := ""
	if container != nil {
		containerID = container.ID
	}
	logrus.Info("containerID " + containerID)
	created := false
	if len(containerID) == 0 {
		newID, createErr := createContainer(client, &config, &hostConfig, imageTag, instance, name, progress)
		if createErr != nil {
			logrus.Error(fmt.Sprintf("fail to create container error :%s", createErr.Error()))
		} else {
			containerID = newID
			created = true
		}
	}
	if len(containerID) == 0 {
		logrus.Error("no container id!")
	}
	logrus.Info(fmt.Sprintf("Starting docker container [%s] docker id [%s] %v", name, containerID, startConfig))

	startErr := client.ContainerStart(context.Background(), containerID, types.ContainerStartOptions{})

	if startErr != nil {
		if created {
			if err1 := removeContainer(client, containerID); err1 != nil {
				logrus.Error(err1)
			}
		}
		logrus.Error(startErr)
	}

	RecordState(client, instance, containerID)
	return nil
}

func GetInstanceAndHost(event *events.Event) (*model.Instance, *model.Host) {

	data := event.Data
	var ihm model.InstanceHostMap
	mapstructure.Decode(data["instanceHostMap"], &ihm)

	var instance model.Instance
	if err := mapstructure.Decode(ihm.Instance, &instance); err != nil {
		panic(err)
	}
	var host model.Host
	if err := mapstructure.Decode(ihm.Host, &host); err != nil {
		panic(err)
	}

	clusterConnection, ok := GetFieldsIfExist(data, "field", "clusterConnection")
	if ok {
		host.Data["clusterConnection"] = clusterConnection.(string)
		if strings.HasPrefix(clusterConnection.(string), "http") {
			caCrt, ok1 := GetFieldsIfExist(event.Data, "field", "caCrt")
			clientCrt, ok2 := GetFieldsIfExist(event.Data, "field", "clientCrt")
			clientKey, ok3 := GetFieldsIfExist(event.Data, "field", "clientKey")
			// what if we miss certs/key? do we have to panic or ignore it?
			if ok1 && ok2 && ok3 {
				host.Data["caCrt"] = caCrt.(string)
				host.Data["clientCrt"] = clientCrt.(string)
				host.Data["clientKey"] = clientKey.(string)
			} else {
				logrus.Error("Missing certs/key for clusterConnection for connection " +
					clusterConnection.(string))
				panic(errors.New("Missing certs/key for clusterConnection for connection " +
					clusterConnection.(string)))
			}
		}
	}
	return &instance, &host
}

func IsInstanceActive(instance *model.Instance, host *model.Host) bool {
	if isNoOp(instance.Data) {
		return true
	}

	client := docker.GetClient(DefaultVersion)
	container := GetContainer(client, instance, false)
	return isRunning(client, container)
}

func isNoOp(data map[string]interface{}) bool {
	b, ok := GetFieldsIfExist(data, "containerNoOpEvent")
	if ok {
		return b.(bool)
	}
	return false
}

func GetContainer(client *client.Client, instance *model.Instance, byAgent bool) *types.Container {
	if instance == nil {
		return nil
	}

	// First look for UUID label directly
	args := filters.NewArgs()
	args.Add("label", fmt.Sprintf("%s=%s", UUIDLabel, instance.UUID))
	options := types.ContainerListOptions{All: true, Filter: args}
	labeledContainers, err := client.ContainerList(context.Background(), options)
	if err == nil && len(labeledContainers) > 0 {
		return &labeledContainers[0]
	}

	// Nest look by UUID using fallback method
	options = types.ContainerListOptions{All: true}
	containerList, err := client.ContainerList(context.Background(), options)
	if err != nil {
		return nil
	}
	container := findFirst(&containerList, func(c *types.Container) bool {
		if getUUID(c) == instance.UUID {
			return true
		}
		return false
	})

	if container != nil {
		return container
	}
	if externalID := instance.ExternalID; externalID != "" {
		container = findFirst(&containerList, func(c *types.Container) bool {
			return idFilter(externalID, c)
		})
	}

	if container != nil {
		return container
	}

	if agentID := instance.AgentID; byAgent {
		container = findFirst(&containerList, func(c *types.Container) bool {
			return agentIDFilter(string(agentID), c)
		})
	}

	return container

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

func getUUID(container *types.Container) string {
	uuid, err := container.Labels[UUIDLabel]
	if err {
		return uuid
	}

	names := container.Names
	if names == nil {
		return fmt.Sprintf("no-uuid-%s", container.ID)
	}

	if strings.HasPrefix(names[0], "/") {
		return names[0][1:]
	}
	return names[0]
}

func findFirst(containers *[]types.Container, f func(*types.Container) bool) *types.Container {
	for _, c := range *containers {
		if f(&c) {
			return &c
		}
	}
	return nil
}

func idFilter(id string, container *types.Container) bool {
	return container.ID == id
}

func agentIDFilter(id string, container *types.Container) bool {
	containerID, ok := container.Labels["io.rancher.container.agent_id"]
	if ok {
		return containerID == id
	}
	return false
}

func RecordState(client *client.Client, instance *model.Instance, dockerID string) {
	if len(dockerID) == 0 {
		container := GetContainer(client, instance, false)
		if container != nil {
			dockerID = container.ID
		}
	}

	if len(dockerID) == 0 {
		return
	}
	contDir := containerStateDir()
	temFilePath := path.Join(contDir, fmt.Sprintf("tmp-%s", dockerID))
	if _, err := os.Stat(temFilePath); err == nil {
		os.Remove(temFilePath)
	}
	filePath := path.Join(contDir, dockerID)
	if _, err := os.Stat(filePath); err == nil {
		os.Remove(filePath)
	}
	if _, err := os.Stat(contDir); err != nil {
		err1 := os.MkdirAll(contDir, 644)
		if err1 != nil {
			logrus.Info("can't create dir")
		}
	}
	data, _ := marshaller.ToString(instance)
	ioutil.WriteFile(temFilePath, data, 0644)
	os.Rename(temFilePath, filePath)
	_, err1 := os.Stat(filePath)
	if err1 != nil {
		logrus.Error(err1)
	}
	logrus.Infof("fileinfo %v", filePath)
}

func createContainer(client *client.Client, config *container.Config, hostConfig *container.HostConfig,
	imageTag string, instance *model.Instance, name string, progress *progress.Progress) (string, error) {
	logrus.Info("Creating docker container from config")
	labels := config.Labels
	if labels["io.rancher.container.pull_image"] == "always" {
		DoInstancePull(&model.ImageParams{
			Image:    instance.Image,
			Tag:      "",
			Mode:     "all",
			Complete: false,
		}, progress)
	}
	// delete(createConfig, "name")
	command := []string{}
	if len(config.Cmd) > 0 {
		command = config.Cmd
	}
	logrus.Info(command)
	config.Image = imageTag

	if vDriver, ok := GetFieldsIfExist(instance.Data, "field", "volumeDriver"); ok {
		hostConfig.VolumeDriver = vDriver.(string)
	}

	containerResponse, err := client.ContainerCreate(context.Background(), config, hostConfig, nil, name)
	logrus.Info(fmt.Sprintf("creating container with name %s", name))
	// if image doesn't exist
	if err != nil {
		logrus.Error(err)
		if strings.Contains(err.Error(), config.Image) {
			pullImage(&instance.Image, progress)
			containerResponse, err1 := client.ContainerCreate(context.Background(), config, hostConfig, nil, name)
			if err1 != nil {
				logrus.Error(fmt.Sprintf("container id %s fail to start", containerResponse.ID))
				return "", err1
			}
			return containerResponse.ID, nil
		}
		return "", err
	}
	logrus.Info(containerResponse.ID)
	return containerResponse.ID, nil
}

func removeContainer(client *client.Client, containerID string) error {
	err := client.ContainerRemove(context.Background(), containerID, types.ContainerRemoveOptions{})
	return err
}

func DoInstancePull(params *model.ImageParams, progress *progress.Progress) (types.ImageInspect, error) {
	client := docker.GetClient(DefaultVersion)

	imageJSON, ok := GetFieldsIfExist(params.Image.Data, "dockerImage")
	if !ok {
		return types.ImageInspect{}, errors.New("field not exist")
	}
	var dockerImage model.DockerImage
	mapstructure.Decode(imageJSON, &dockerImage)
	existing, _, err := client.ImageInspectWithRaw(context.Background(), dockerImage.ID, false)
	if err != nil {
		return types.ImageInspect{}, err
	}
	if params.Mode == "cached" {
		return existing, nil
	}
	if params.Complete {
		var err1 error
		_, err1 = client.ImageRemove(context.Background(), dockerImage.ID, types.ImageRemoveOptions{})
		return types.ImageInspect{}, err1
	}

	imagePull(params, progress)

	if len(params.Tag) > 0 {
		imageInfo := parseRepoTag(dockerImage.FullName)
		repoTag := fmt.Sprintf("%s:%s", imageInfo["repo"], imageInfo["tag"]+params.Tag)
		client.ImageTag(context.Background(), dockerImage.ID, repoTag)
	}

	inspect, _, err2 := client.ImageInspectWithRaw(context.Background(), dockerImage.ID, false)
	return inspect, err2
}

func createContainerConfig(imageTag string, command strslice.StrSlice, createConfig map[string]interface{}) *container.Config {
	createConfig["image"] = imageTag
	var config container.Config
	err := mapstructure.Decode(createConfig, &config)
	res, _ := json.Marshal(config)
	logrus.Infof("config created %v", string(res))
	if err != nil {
		panic(err)
	}
	return &config
}

func getImageTag(instance *model.Instance) (string, error) {
	var dockerImage model.DockerImage
	mapstructure.Decode(instance.Image.Data["dockerImage"], &dockerImage)
	imageName := dockerImage.FullName
	if imageName == "" {
		return "", errors.New("Can not start container with no image")
	}
	return imageName, nil
}

func isTrue(instance *model.Instance, field string) bool {
	value, ok := GetFieldsIfExist(instance.Data, "fields", field)
	if ok {
		return value.(bool)
	}
	return false
}

func setupSimpleConfigFields(config map[string]interface{}, instance *model.Instance, fields []model.Tuple) {
	for _, tuple := range fields {
		src := tuple.Src
		dest := tuple.Dest
		srcObj, ok := GetFieldsIfExist(instance.Data, "field", src)
		if !ok {
			break
		}
		config[dest] = unwrap(&srcObj)
	}
}

func setupDNSSearch(startConfig map[string]interface{}, instance *model.Instance) {
	systemCon := instance.SystemContainer
	if len(systemCon) > 0 {
		return
	}
	// if only rancher search is specified,
	// prepend search with params read from the system
	allRancher := true
	dnsSearch, ok2 := startConfig["dnsSearch"].([]string)
	if ok2 {
		logrus.Info("hello")
		if dnsSearch == nil || len(dnsSearch) == 0 {
			return
		}
		for _, search := range dnsSearch {
			if strings.HasSuffix(search, "rancher.internal") {
				continue
			}
			allRancher = false
			break
		}
	} else {
		return
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
				if !searchInList(s, search) {
					dnsSearch = append(dnsSearch, search)
				}
			}
			startConfig["dnsSearch"] = dnsSearch
		}
	}

}

func setupLogging(startConfig map[string]interface{}, instance *model.Instance) {
	logConfig, ok := startConfig["logConfig"]
	if !ok {
		return
	}
	driver, ok := logConfig.(map[string]interface{})["driver"].(string)

	if ok {
		delete(startConfig["logConfig"].(map[string]interface{}), "driver")
		startConfig["logConfig"].(map[string]interface{})["type"] = driver
	}

	for _, value := range []string{"type", "config"} {
		bad := true
		obj, ok := startConfig["logConfig"].(map[string]interface{})[value]
		if ok && obj != nil {
			bad = false
			startConfig["logConfig"].(map[string]interface{})[value] = unwrap(&obj)
		}
		if _, ok1 := startConfig["logConfig"]; bad && ok1 {
			delete(startConfig, "logConfig")
		}
	}

}

func setupHostname(createConfig map[string]interface{}, instance *model.Instance) {
	name := instance.Hostname
	if len(name) > 0 {
		createConfig["hostname"] = name
	}
}

func setupCommand(createConfig map[string]interface{}, instance *model.Instance) {
	command, ok := GetFieldsIfExist(instance.Data, "fields", "command")
	if !ok {
		return
	}
	commands, ok := command.([]interface{})
	if !ok {
		setupLegacyCommand(createConfig, instance, command.(string))
	} else {
		createConfig["cmd"] = commands
	}
}

func setupPorts(createConfig map[string]interface{}, instance *model.Instance,
	startConfig map[string]interface{}) {
	ports := []model.Port{}
	exposedPorts := map[nat.Port]struct{}{}
	bindings := nat.PortMap{}
	if instance.Ports != nil && len(instance.Ports) > 0 {
		for _, port := range instance.Ports {
			ports = append(ports, model.Port{PrivatePort: port.PrivatePort, Protocol: port.Protocol})
			if port.PrivatePort != 0 {
				bind := nat.Port(fmt.Sprintf("%v/%v", port.PrivatePort, port.Protocol))
				logrus.Info(bind)
				bindAddr := ""
				if bindAddress, ok := GetFieldsIfExist(port.Data, "fields", "bindAddress"); ok {
					bindAddr = bindAddress.(string)
				}
				if _, ok := bindings[bind]; !ok {
					bindings[bind] = []nat.PortBinding{nat.PortBinding{HostIP: bindAddr,
						HostPort: convertPortToString(port.PublicPort)}}
				} else {
					bindings[bind] = append(bindings[bind], nat.PortBinding{HostIP: bindAddr,
						HostPort: convertPortToString(port.PublicPort)})
				}
				exposedPorts[bind] = struct{}{}
			}

		}
	}

	createConfig["exposedPorts"] = exposedPorts

	if len(bindings) > 0 {
		startConfig["portbindings"] = bindings
		logrus.Infof("binding map %v", bindings)
	}

}

func setupVolumes(createConfig map[string]interface{}, instance *model.Instance,
	startConfig map[string]interface{}, client *client.Client) {
	if volumes, ok := GetFieldsIfExist(instance.Data, "fields", "dataVolumes"); ok {
		volumes := volumes.([]interface{})
		volumesMap := map[string]interface{}{}
		binds := []string{}
		if len(volumes) > 0 {
			for _, volume := range volumes {
				volume := volume.(string)
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
			createConfig["volumes"] = volumesMap
			startConfig["binds"] = binds
		}
	}

	containers := []string{}
	if vfsList := instance.DataVolumesFromContainers; vfsList != nil {
		for _, vfs := range vfsList {
			var in model.Instance
			mapstructure.Decode(vfs, &in)
			container := GetContainer(client, &in, false)
			if container != nil {
				containers = append(containers, container.ID)
			}
		}
		logrus.Infof("volumes From %v", containers)
		if containers != nil && len(containers) > 0 {
			startConfig["volumesFrom"] = containers
		}
	}

	if vMounts := instance.VolumesFromDataVolumeMounts; len(vMounts) > 0 {
		for vMount := range vMounts {
			var volume model.Volume
			err := mapstructure.Decode(vMount, &volume)
			storagePool := model.StoragePool{}
			progress := progress.Progress{}
			if err != nil {
				if !IsVolumeActive(&volume, &storagePool) {
					DoVolumeActivate(&volume, &storagePool, &progress)
				}
			}
		}
	}
}

func setupLinks(startConfig map[string]interface{}, instance *model.Instance) {
	links := []string{}

	if instance.InstanceLinks == nil {
		return
	}
	for _, link := range instance.InstanceLinks {
		if link.TargetInstance.UUID != "" {
			logrus.Info("hello")
			linkStr := fmt.Sprintf("%s:%s", link.TargetInstance.UUID, link.LinkName)
			links = append(links, linkStr)
		}
	}
	logrus.Infof("links info %v", links)
	startConfig["links"] = links

}

func setupNetworking(instance *model.Instance, host *model.Host,
	createConfig map[string]interface{}, startConfig map[string]interface{}) {
	client := docker.GetClient(DefaultVersion)
	portsSupported, hostnameSupported := setupNetworkMode(instance, client, createConfig, startConfig)
	setupMacAndIP(instance, createConfig, portsSupported, hostnameSupported)
	setupPortsNetwork(instance, createConfig, startConfig, portsSupported)
	setupLinksNetwork(instance, createConfig, startConfig)
	setupIpsec(instance, host, createConfig, startConfig)
	setupDNS(instance)
}

func flagSystemContainer(instance *model.Instance, createConfig map[string]interface{}) {
	if len(instance.SystemContainer) > 0 {
		addLabel(createConfig, map[string]string{"io.rancher.container.system": instance.SystemContainer})
	}
}

func setupProxy(instance *model.Instance, createConfig map[string]interface{}) {
	if len(instance.SystemContainer) > 0 {
		if !hasKey(createConfig, "environment") {
			createConfig["environment"] = map[string]interface{}{}
		}
		for _, i := range []string{"http_proxy", "https_proxy", "NO_PROXY"} {
			createConfig["enviroment"].(map[string]interface{})[i] = os.Getenv(i)
		}
	}
}

func setupCattleConfigURL(instance *model.Instance, createConfig map[string]interface{}) {
	if instance.AgentID == 0 && !hasLabel(instance) {
		return
	}

	if !hasKey(createConfig, "labels") {
		createConfig["labels"] = make(map[string]string)
	}
	addLabel(createConfig, map[string]string{"io.rancher.container.agent_id": strconv.Itoa(instance.AgentID)})

	url := configURL()

	if len(url) > 0 {
		parsed, err := urls.Parse(url)

		if err != nil {
			logrus.Error(err)
			panic(err)
		} else {
			if parsed.Host == "localhost" {
				port := apiProxyListenPort()
				addToEnv(createConfig, map[string]string{
					"CATTLE_AGENT_INSTANCE":    "true",
					"CATTLE_CONFIG_URL_SCHEME": parsed.Scheme,
					"CATTLE_CONFIG_URL_PATH":   parsed.Path,
					"CATTLE_CONFIG_URL_PORT":   string(port),
				})
			} else {
				addToEnv(createConfig, map[string]string{
					"CATTLE_CONFIG_URL": url,
					"CATTLE_URL":        url,
				})
			}
		}
	}
}

func setupDeviceOptions(hostConfig *container.HostConfig, instance *model.Instance) {

	if deviceOptions, ok := GetFieldsIfExist(instance.Data, "fields", "blkioDeviceOptions"); ok {
		logrus.Info("hello")
		blkioWeightDevice := []*blkiodev.WeightDevice{}
		blkioDeviceReadIOps := []*blkiodev.ThrottleDevice{}
		blkioDeviceWriteBps := []*blkiodev.ThrottleDevice{}
		blkioDeviceReadBps := []*blkiodev.ThrottleDevice{}
		blkioDeviceWriteIOps := []*blkiodev.ThrottleDevice{}

		deviceOptions := deviceOptions.(map[string]interface{})
		for dev, options := range deviceOptions {
			if dev == "DEFAULT_DICK" {
				//dev = host_info.Get_default_disk()
				if len(dev) == 0 {
					logrus.Warn(fmt.Sprintf("Couldn't find default device. Not setting device options: %s", options))
					continue
				}
			}
			options := options.(map[string]interface{})
			for key, value := range options {
				value := value.(float64)
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

func setupLegacyCommand(createConfig map[string]interface{}, instance *model.Instance, command string) {
	// This can be removed shortly once cattle removes
	// commandArgs
	logrus.Info("set up command")
	if len(command) == 0 || len(strings.TrimSpace(command)) == 0 {
		return
	}
	commandArgs := []string{}
	if value, ok := GetFieldsIfExist(instance.Data, "fields", "commandArgs"); ok {
		for _, v := range value.([]interface{}) {
			commandArgs = append(commandArgs, v.(string))
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
		createConfig["cmd"] = commands
	}
}

func createHostConfig(startConfig map[string]interface{}) container.HostConfig {
	var hostConfig container.HostConfig
	err := mapstructure.Decode(startConfig, &hostConfig)
	if err == nil {
		return hostConfig
	}
	return container.HostConfig{}
}

func DeleteContainer(name string) {
	client := docker.GetClient(DefaultVersion)
	containerList, _ := client.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	for _, container := range containerList {
		found := false
		labels := container.Labels
		if labels["io.rancher.container.uuid"] == name[1:] {
			found = true
		}
		for _, containerName := range container.Names {
			if name == containerName {
				found = true
				break
			}
		}
		if found {
			logrus.Infof("killing the container %v", container.ID)
			killErr := client.ContainerKill(context.Background(), container.ID, "KILL")
			if killErr == nil {
				logrus.Infof("container %v killed", container.ID)
			} else {
				logrus.Error(killErr)
			}
			logrus.Infof("removing container %v", container.ID)
			rmErr := client.ContainerRemove(context.Background(), container.ID, types.ContainerRemoveOptions{})
			if rmErr == nil {
				logrus.Infof("container %v removed", container.ID)
			} else {
				logrus.Error(rmErr)
			}
			removeStateFile(container.ID)
			/*
				for i := 0; i < 10; i++ {
					inspect, err := client.ContainerInspect(context.Background(), container.ID)
					if err == nil && inspect.State.Pid == 0 {
						break
					}
				}
			*/
		}
	}
}

func removeStateFile(id string) {
	if len(id) > 0 {
		contDir := containerStateDir()
		filePath := path.Join(contDir, id)
		if _, err := os.Stat(filePath); err == nil {
			os.Remove(filePath)
		}
	}
}

func DoInstanceDeactivate(instance *model.Instance, progress *progress.Progress) error {
	if isNoOp(instance.Data) {
		return nil
	}

	client := docker.GetClient(DefaultVersion)
	timeout := 10
	if value, ok := GetFieldsIfExist(instance.ProcessData, "timeout"); ok {
		timeout = int(value.(float64))
	}
	time := time.Duration(timeout)
	container := GetContainer(client, instance, false)
	client.ContainerStop(context.Background(), container.ID, &time)
	container = GetContainer(client, instance, false)
	if !isStopped(client, container) {
		client.ContainerKill(context.Background(), container.ID, "KILL")
	}
	if !isStopped(client, container) {
		return fmt.Errorf("Filed to stop container %v", instance.UUID)
	}
	logrus.Infof("container id %v deactivated", container.ID)
	return nil
}

func isStopped(client *client.Client, container *types.Container) bool {
	return !isRunning(client, container)
}

func IsInstanceInactive(instance *model.Instance) bool {
	if isNoOp(instance.Data) {
		return true
	}

	client := docker.GetClient(DefaultVersion)
	container := GetContainer(client, instance, false)
	return isStopped(client, container)
}

func DoInstanceForceStop(request *model.InstanceForceStop) error {
	client := docker.GetClient(DefaultVersion)
	time := time.Duration(0)
	stopErr := client.ContainerStop(context.Background(), request.ID, &time)
	if stopErr != nil {
		logrus.Error(stopErr)
		return stopErr
	}
	logrus.Infof("container id %v is forced to be stopped", request.ID)
	return nil
}

func DoInstanceInspect(inspect *model.InstanceInspect) (types.ContainerJSON, error) {
	client := docker.GetClient(DefaultVersion)
	containerID := inspect.ID
	if containerID == "" {
		return types.ContainerJSON{}, fmt.Errorf("container with id [%v] not found", containerID)
	}
	containerList, _ := client.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	result := findFirst(&containerList, func(c *types.Container) bool {
		return idFilter(containerID, c)
	})
	if result == nil {
		name := fmt.Sprintf("/%s", inspect.Name)
		result = findFirst(&containerList, func(c *types.Container) bool {
			return nameFilter(name, c)
		})
	}
	if result != nil {
		logrus.Infof("start inspecting container with id [%s]", result.ID)
		inspectResp, err := client.ContainerInspect(context.Background(), result.ID)
		if err != nil {
			logrus.Error(err)
			return types.ContainerJSON{}, err
		}
		logrus.Infof("container with id [%s] inspected", result.ID)
		return inspectResp, nil
	}
	return types.ContainerJSON{}, fmt.Errorf("container with id [%v] not found", containerID)
}

func nameFilter(name string, container *types.Container) bool {
	names := container.Names
	if names == nil || len(names) == 0 {
		return false
	}
	found := false
	for _, n := range names {
		if strings.HasSuffix(n, name) {
			found = true
			break
		}
	}
	return found
}

func IsInstanceRemoved(instance *model.Instance) bool {
	client := docker.GetClient(DefaultVersion)
	container := GetContainer(client, instance, false)
	return container == nil
}

func DoInstanceRemove(instance *model.Instance, progress *progress.Progress) error {
	client := docker.GetClient(DefaultVersion)
	container := GetContainer(client, instance, false)
	if container == nil {
		return errors.New("container not found")
	}
	return removeContainer(client, container.ID)
}

func PurgeState(instance *model.Instance) {
	client := docker.GetClient(DefaultVersion)
	container := GetContainer(client, instance, false)
	if container == nil {
		return
	}
	dockerID := container.ID
	contDir := containerStateDir()
	files := []string{path.Join(contDir, "tmp-"+dockerID), path.Join(contDir, dockerID)}
	for _, f := range files {
		if _, err := os.Stat(f); err == nil {
			os.Remove(f)
		}
	}
}

func getInstanceHostMapData(event *revents.Event) map[string]interface{} {
	instance, _ := GetInstanceAndHost(event)
	client := docker.GetClient(DefaultVersion)
	var inspect types.ContainerJSON
	container := GetContainer(client, instance, false)
	logrus.Infof("container structure %v", container)
	dockerPorts := []string{}
	dockerIP := ""
	dockerMounts := []types.MountPoint{}
	if container != nil {
		logrus.Info(container.ID)
		inspect, _ = client.ContainerInspect(context.Background(), container.ID)
		dockerMounts = getMountData(container.ID)
		dockerIP = inspect.NetworkSettings.IPAddress
		if container.Ports != nil && len(container.Ports) > 0 {
			for _, port := range container.Ports {
				privatePort := fmt.Sprintf("%v/%v", port.PrivatePort, port.Type)
				portSpec := privatePort
				bindAddr := ""
				if port.IP != "" {
					bindAddr = fmt.Sprintf("%s:", port.IP)
				}
				publicPort := ""
				if port.PublicPort > 0 {
					publicPort = fmt.Sprintf("%v:", port.PublicPort)
				} else if port.IP != "" {
					publicPort = ":"
				}
				portSpec = bindAddr + publicPort + portSpec
				dockerPorts = append(dockerPorts, portSpec)
			}
		}
	}
	update := map[string]interface{}{
		"instance": map[string]interface{}{
			"+data": map[string]interface{}{
				"dockerContainer": container,
				"dockerInspect":   inspect,
				"+fields": map[string]interface{}{
					"dockerHostIp": DockerHostIP(),
					"dockerPorts":  dockerPorts,
					"dockerIp":     dockerIP,
				},
			},
		},
	}
	if container != nil {
		update["instance"].(map[string]interface{})["externalId"] = container.ID
	}
	if dockerMounts != nil {
		update["instance"].(map[string]interface{})["+data"].(map[string]interface{})["dockerMounts"] = dockerMounts
	}
	return update
}

func getMountData(containerID string) []types.MountPoint {
	client := docker.GetClient(DefaultVersion)
	inspect, _ := client.ContainerInspect(context.Background(), containerID)
	return inspect.Mounts
}

func setupResource(fields map[string]interface{}, hostConfig *container.HostConfig) {
	var resource container.Resources
	mapstructure.Decode(fields, &resource)
	hostConfig.Resources = resource
	setupHostConfig(fields, hostConfig)
}

func setupHostConfig(fields map[string]interface{}, hostConfig *container.HostConfig) {
	for key, value := range fields {
		switch key {
		case "extraHosts":
			for _, singleValue := range value.([]interface{}) {
				if str := InterfaceToString(singleValue); str != "" {
					hostConfig.ExtraHosts = append(hostConfig.ExtraHosts, str)
				}
			}
			break
		case "pidMode":
			hostConfig.PidMode = container.PidMode(InterfaceToString(value))
			break
		case "logConfig":
			value, ok := value.(map[string]interface{})
			if !ok {
				return
			}
			hostConfig.LogConfig.Type = InterfaceToString(value["driver"])
			hostConfig.LogConfig.Config = map[string]string{}
			if value["config"] != nil {
				for key1, value1 := range value["config"].(map[string]interface{}) {
					hostConfig.LogConfig.Config[key1] = InterfaceToString(value1)
				}
			}
			break
		case "securityOpt":
			for _, singleValue := range value.([]interface{}) {
				if str := InterfaceToString(singleValue); str != "" {
					hostConfig.SecurityOpt = append(hostConfig.SecurityOpt, str)
				}
			}
			break
		case "devices":
			hostConfig.Devices = []container.DeviceMapping{}
			for _, singleValue := range value.([]interface{}) {
				str := InterfaceToString(singleValue)
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
			for _, singleValue := range value.([]interface{}) {
				if str := InterfaceToString(singleValue); str != "" {
					hostConfig.DNS = append(hostConfig.DNS, str)
				}
			}
			break
		case "dnsSearch":
			for _, singleValue := range value.([]interface{}) {
				if str := InterfaceToString(singleValue); str != "" {
					hostConfig.DNSSearch = append(hostConfig.DNSSearch, str)
				}
			}
			break
		case "capAdd":
			for _, singleValue := range value.([]interface{}) {
				if str := InterfaceToString(singleValue); str != "" {
					hostConfig.CapAdd = append(hostConfig.CapAdd, str)
				}
			}
			break
		case "capDrop":
			for _, singleValue := range value.([]interface{}) {
				if str := InterfaceToString(singleValue); str != "" {
					hostConfig.CapDrop = append(hostConfig.CapDrop, str)
				}
			}
			break
		case "restartPolicy":
			value := value.(map[string]interface{})
			hostConfig.RestartPolicy.Name = InterfaceToString(value["name"])
			if _, ok := value["maximumRetryCount"].(float64); ok {
				hostConfig.RestartPolicy.MaximumRetryCount = int(value["maximumRetryCount"].(float64))
			}
			break
		case "cpuShares":
			if _, ok := value.(float64); ok {
				hostConfig.CPUShares = int64(value.(float64))
			}
		}
	}
}

func setupConfig(fields map[string]interface{}, config *container.Config) {
	for key, value := range fields {
		switch key {
		case "workingDir":
			config.WorkingDir = InterfaceToString(value)
			break
		case "entryPoint":
			for _, singleValue := range value.([]interface{}) {
				if str := InterfaceToString(singleValue); str != "" {
					config.Entrypoint = append(config.Entrypoint, str)
				}
			}
			break
		case "tty":
			config.Tty = value.(bool)
			break
		case "stdinOpen":
			config.OpenStdin = value.(bool)
			break
		case "domainName":
			config.Domainname = value.(string)
			break
		case "labels":
			for k, v := range value.(map[string]interface{}) {
				str := InterfaceToString(v)
				config.Labels[k] = str
			}
		}

	}
}
