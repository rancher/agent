package utils

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/container"
	"github.com/docker/engine-api/types/filters"
	"github.com/docker/go-connections/nat"
	"github.com/mitchellh/mapstructure"
	"github.com/rancher/agent/handlers/dockerClient"
	"github.com/rancher/agent/handlers/marshaller"
	"github.com/rancher/agent/handlers/progress"
	"github.com/rancher/agent/model"
	"github.com/rancher/go-machine-service/events"
	"golang.org/x/net/context"
	urls "net/url"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
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

	clusterConnection, ok := getFieldsIfExist(data, "field", "clusterConnection")
	if ok {
		logrus.Debugf("clusterConnection = %s", clusterConnection.(string))
		host.Data["clusterConnection"] = clusterConnection.(string)
		if strings.HasPrefix(clusterConnection.(string), "http") {
			caCrt, ok1 := getFieldsIfExist(event.Data, "field", "caCrt")
			clientCrt, ok2 := getFieldsIfExist(event.Data, "field", "clientCrt")
			clientKey, ok3 := getFieldsIfExist(event.Data, "field", "clientKey")
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

	client := dockerClient.GetClient(DefaultVersion)
	container := getContainer(client, instance, false)
	return isRunning(client, container)
}

func isNoOp(data map[string]interface{}) bool {
	b, ok := getFieldsIfExist(data, "containerNoOpEvent")
	if ok {
		return b.(bool)
	}
	return false
}

func getContainer(client *client.Client, instance *model.Instance, byAgent bool) *types.Container {
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
	if len(dockerID) > 0 {
		container := getContainer(client, instance, false)
		if container != nil {
			dockerID = container.ID
		}
	}

	if len(dockerID) == 0 {
		return
	}
	contDir := containerStateDir()
	temFilePath := path.Join(contDir, fmt.Sprintf("tmp-%s", dockerID))
	if _, err := os.Stat(temFilePath); err != nil {
		os.Remove(temFilePath)
	}
	filePath := path.Join(contDir, dockerID)
	if _, err := os.Stat(filePath); err != nil {
		os.Remove(filePath)
	}
	if _, err := os.Stat(contDir); err != nil {
		os.Mkdir(contDir, 777)
	}
	file, _ := os.Open(temFilePath)
	data, _ := marshaller.ToString(instance)
	defer file.Close()
	file.Write(data)

	os.Rename(temFilePath, filePath)
}

func DoInstanceActivate(instance *model.Instance, host *model.Host, progress *progress.Progress) error {
	if isNoOp(instance.Data) {
		return nil
	}

	client := dockerClient.GetClient(DefaultVersion)

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
			logrus.Info(id)
			_, err1 := client.ContainerInspect(context.Background(), id)
			if err1 != nil {
				logrus.Info("container doesn't exists")
				name = id
				logrus.Info(id)
			} else {
				logrus.Info("container exists")
			}
		}
	}
	logrus.Info(name)
	var createConfig = map[string]interface{}{
		"name":   name,
		"detach": true,
	}

	var startConfig = map[string]interface{}{
		"publish_all_ports": false,
		"privileged":        isTrue(instance, "privileged"),
		"read_only":         isTrue(instance, "readOnly"),
	}

	// These _setupSimpleConfigFields calls should happen before all
	// other config because they stomp over config fields that other
	// setup methods might append to. Example: the environment field
	setupSimpleConfigFields(createConfig, instance,
		CreateConfigFields)

	setupSimpleConfigFields(startConfig, instance,
		StartConfigFields)

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

	createConfig["hostConfig"] = createHostConfig(startConfig)

	setupDeviceOptions(createConfig["hostConfig"].(container.HostConfig), instance)

	//debug
	var config container.Config
	var hostConfig container.HostConfig
	mapstructure.Decode(createConfig, &config)
	mapstructure.Decode(createHostConfig(startConfig), &hostConfig)
	s1, _ := marshaller.ToString(config)
	s2, _ := marshaller.ToString(hostConfig)
	logrus.Info(fmt.Sprintf("container configuration %s", string(s1)))
	logrus.Info(fmt.Sprintf("container host configuration %s", string(s2)))

	container := getContainer(client, instance, false)
	containerID := ""
	if container != nil {
		containerID = container.ID
	}
	logrus.Info("containerID " + containerID)
	created := false
	if len(containerID) == 0 {
		newID, createErr := createContainer(client, createConfig, imageTag, instance, name, progress)
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

func createContainer(client *client.Client, createConfig map[string]interface{},
	imageTag string, instance *model.Instance, name string, progress *progress.Progress) (string, error) {
	logrus.Info("Creating docker container [%s] from config")
	// debug
	logrus.Debug("debug")
	labels := createConfig["labels"]
	if labels.(map[string]string)["io.rancher.container.pull_image"] == "always" {
		doInstancePull(&model.ImageParams{
			Image:    instance.Image,
			Tag:      "",
			Mode:     "all",
			Complete: false,
		}, progress)
	}
	delete(createConfig, "name")
	command := ""
	if createConfig["command"] != nil {
		command = createConfig["command"].(string)
	}
	logrus.Info(command)
	delete(createConfig, "command")
	config := createContainerConfig(imageTag, command, createConfig)
	hostConfig := createConfig["hostConfig"].(container.HostConfig)

	if vDriver, ok := getFieldsIfExist(instance.Data, "field", "volumeDriver"); ok {
		hostConfig.VolumeDriver = vDriver.(string)
	}

	containerResponse, err := client.ContainerCreate(context.Background(), config, &hostConfig, nil, name)
	logrus.Info(fmt.Sprintf("creating container with name %s", name))
	// if image doesn't exist
	if err != nil {
		logrus.Error(err)
		if strings.Contains(err.Error(), config.Image) {
			pullImage(instance.Image, progress)
			containerResponse, err1 := client.ContainerCreate(context.Background(), config, &hostConfig, nil, name)
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

func doInstancePull(params *model.ImageParams, progress *progress.Progress) (types.ImageInspect, error) {
	client := dockerClient.GetClient(DefaultVersion)

	imageJSON, ok := getFieldsIfExist(params.Image.Data, "dockerImage")
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

func createContainerConfig(imageTag string, command string, createConfig map[string]interface{}) *container.Config {
	if len(command) > 0 {
		createConfig["cmd"] = []string{command}
	}
	createConfig["image"] = imageTag
	var config container.Config
	err := mapstructure.Decode(createConfig, &config)
	res, _ := json.Marshal(config)
	logrus.Info(string(res))
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
	_, ok := getFieldsIfExist(instance.Data, field)
	return ok
}

func setupSimpleConfigFields(config map[string]interface{}, instance *model.Instance, fields []model.Tuple) {
	for _, tuple := range fields {
		src := tuple.Src
		dest := tuple.Dest
		srcObj, ok := getFieldsIfExist(instance.Data, "field", src)
		if !ok {
			break
		}
		config[dest] = unwrap(&srcObj)
	}
}

func setupDNSSearch(startConfig map[string]interface{}, instance *model.Instance) {
	containerID := instance.SystemContainer
	if len(containerID) == 0 {
		return
	}
	// if only rancher search is specified,
	// prepend search with params read from the system
	allRancher := true
	dnsSearch, ok2 := startConfig["dnsSearch"].([]string)
	if ok2 {
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
	command, ok := getFieldsIfExist(instance.Data, "field", "command")
	if !ok {
		return
	}
	switch command.(type) {
	case string:
		setupLegacyCommand(createConfig, instance, command.(string))
	default:
		if command != nil {
			createConfig["command"] = command
		}
	}
}

func setupPorts(createConfig map[string]interface{}, instance *model.Instance,
	startConfig map[string]interface{}) {
	ports := []model.Port{}
	bindings := nat.PortMap{}
	if instance.Ports != nil && len(instance.Ports) > 0 {
		for _, port := range instance.Ports {
			ports = append(ports, model.Port{PrivatePort: port.PrivatePort, Protocol: port.Protocol})
			if port.PrivatePort != 0 {
				bind := nat.Port(fmt.Sprintf("%d/%s", port.PrivatePort, port.Protocol))
				logrus.Info(bind)
				bindAddr := ""
				if bindAddress, ok := getFieldsIfExist(port.Data, "fields", "bindAddress"); ok {
					bindAddr = bindAddress.(string)
				}
				if _, ok := bindings[bind]; !ok {
					bindings[bind] = []nat.PortBinding{nat.PortBinding{HostIP: bindAddr,
						HostPort: convertPortToString(port.PublicPort)}}
				} else {
					bindings[bind] = append(bindings[bind], nat.PortBinding{HostIP: bindAddr,
						HostPort: convertPortToString(port.PublicPort)})
				}
			}

		}
	}

	if len(ports) > 0 {
		createConfig["port"] = ports
	}

	if len(bindings) > 0 {
		startConfig["portbindings"] = bindings
	}

}

func setupVolumes(createConfig map[string]interface{}, instance *model.Instance,
	startConfig map[string]interface{}, client *client.Client) {
	if volumes, ok := getFieldsIfExist(instance.Data, "field", "dataVolumes"); ok {
		volumes := volumes.([]string)
		volumesMap := make(map[string]interface{})
		bindsMap := make(map[string]interface{})
		if len(volumes) > 0 {
			for _, volume := range volumes {
				parts := strings.SplitAfterN(volume, ":", 3)
				if len(parts) == 1 {
					volumesMap[parts[0]] = make(map[string]interface{})
				} else {
					mode := ""
					if len(parts) == 3 {
						mode = parts[2]
					} else {
						mode = "rw"
					}
					bind := struct {
						Binds string
						Mode  string
					}{
						parts[1],
						mode,
					}
					bindsMap[parts[0]] = bind
				}
			}
			createConfig["volumes"] = volumesMap
			startConfig["binds"] = bindsMap
		}
	}

	containers := []string{}
	if vfsList := instance.DataVolumesFromContainers; vfsList != nil {
		for _, vfs := range vfsList {
			var in model.Instance
			mapstructure.Decode(vfs, &in)
			container := getContainer(client, &in, false)
			if container != nil {
				containers = append(containers, container.ID)
			}
		}
		if containers != nil && len(containers) > 0 {
			startConfig["volumes_from"] = containers
		}
	}

	if vMounts := instance.DataVolumesFromContainers; len(vMounts) > 0 {
		for vMount := range vMounts {
			var volume model.Volume
			err := mapstructure.Decode(vMount, &volume)
			if err != nil {
				if !isVolumeActive(volume) {
					doVolumeActivate(volume)
				}
			} else {
				panic(err)
			}
		}
	}
}

func setupLinks(startConfig map[string]interface{}, instance *model.Instance) {
	links := make(map[string]interface{})

	if instance.InstanceLinks == nil {
		return
	}

	for _, link := range instance.InstanceLinks {
		if link.TargetInstanceID != "" {
			links[link.TargetInstance.UUID] = link.LinkName
		}
	}
	startConfig["links"] = links

}

func setupNetworking(instance *model.Instance, host *model.Host,
	createConfig map[string]interface{}, startConfig map[string]interface{}) {
	client := dockerClient.GetClient(DefaultVersion)
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

	if hasKey(createConfig, "labels") {
		createConfig["labels"] = make(map[string]string)
	}
	createConfig["labels"].(map[string]string)["io.rancher.container.agent_id"] = strconv.Itoa(instance.AgentID)

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

func setupDeviceOptions(config container.HostConfig, instance *model.Instance) {
	optionConfigs := []model.OptionConfig{
		model.OptionConfig{
			Key:         "readIops",
			DevList:     []map[string]string{},
			DockerField: "BlkioDeviceReadIOps",
			Field:       "Rate",
		},
		model.OptionConfig{
			Key:         "writeIops",
			DevList:     []map[string]string{},
			DockerField: "BlkioDeviceWriteIOps",
			Field:       "Rate",
		},
		model.OptionConfig{
			Key:         "readBps",
			DevList:     []map[string]string{},
			DockerField: "BlkioDeviceReadBps",
			Field:       "Rate",
		},
		model.OptionConfig{
			Key:         "writeBps",
			DevList:     []map[string]string{},
			DockerField: "BlkioDeviceWriteBps",
			Field:       "Rate",
		},
		model.OptionConfig{
			Key:         "weight",
			DevList:     []map[string]string{},
			DockerField: "BlkioWeightDevice",
			Field:       "Weight",
		},
	}

	if deviceOptions, ok := getFieldsIfExist(instance.Data, "field", "blkioDeviceOptions"); ok {
		deviceOptions := deviceOptions.(map[string]map[string]string)
		for dev, options := range deviceOptions {
			if dev == "DEFAULT_DICK" {
				//dev = host_info.Get_default_disk()
				if len(dev) == 0 {
					logrus.Warn(fmt.Sprintf("Couldn't find default device. Not setting device options: %s", options))
					continue
				}
			}
			for _, oC := range optionConfigs {
				key, devList, field := oC.Key, oC.DevList, oC.Field
				if hasKey(options, key) && len(options[key]) > 0 {
					value := options[key]
					devList = append(devList, map[string]string{"Path": dev, field: value})
				}
			}
		}
	}
	/*
		for _, oC := range optionConfigs {
			devList, docker_field := oC.DevList, oC.DockerField
			if len(devList) >0 {
				config.D = devList
			}
		}
	*/

}

func setupLegacyCommand(createConfig map[string]interface{}, instance *model.Instance, command string) {
	// This can be removed shortly once cattle removes
	// commandArgs
	if len(command) > 0 || len(strings.TrimSpace(command)) == 0 {
		return
	}

	commandArgs := []string{}
	if value := instance.CommandArgs; value != nil {
		commandArgs = value
	}
	commands := []string{}
	if commandArgs != nil && len(commandArgs) > 0 {
		commands = append(commands, command)
		for _, value := range commandArgs {
			commands = append(commands, value)
		}
	}

	if len(commands) > 0 {
		createConfig["command"] = commands
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
