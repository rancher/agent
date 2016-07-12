package utils

import (
	"github.com/Sirupsen/logrus"
	"strings"
	"github.com/docker/engine-api/client"
	"fmt"
	"golang.org/x/net/context"
	"errors"
	"regexp"
	"os"
	"bufio"
	"github.com/mitchellh/mapstructure"
	urls "net/url"
	"strconv"
	"github.com/rancher/agent/model"
	"github.com/rancher/agent/handlers/progress"
	"github.com/docker/engine-api/types"
	"github.com/rancher/agent/handlers/docker_client"
	"github.com/rancher/go-machine-service/events"
	"github.com/docker/engine-api/types/filters"
	"github.com/docker/engine-api/types/container"
)

var CREATE_CONFIG_FIELDS = []model.Tuple{
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

var START_CONFIG_FIELDS = []model.Tuple{
	model.Tuple{Src: "capAdd", Dest: "cap_add"},
	model.Tuple{Src: "capDrop", Dest: "cap_drop"},
	model.Tuple{Src: "dnsSearch", Dest: "dns_search"},
	model.Tuple{Src: "dns", Dest: "dns"},
	model.Tuple{Src: "extraHosts", Dest: "extra_hosts"},
	model.Tuple{Src: "publishAllPorts", Dest: "publish_all_ports"},
	model.Tuple{Src: "lxcConf", Dest: "lxc_conf"},
	model.Tuple{Src: "logConfig", Dest: "log_config"},
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

	clusterConnection, ok := get_fields_if_exist(data, "field", "clusterConnection")
	if ok {
		logrus.Debugf("clusterConnection = %s", clusterConnection.(string))
		host.Data["clusterConnection"] = clusterConnection.(string)
		if strings.HasPrefix(clusterConnection.(string), "http") {
			caCrt, ok1 := get_fields_if_exist(event.Data, "field", "caCrt")
			clientCrt, ok2 :=  get_fields_if_exist(event.Data, "field", "clientCrt")
			clientKey, ok3 :=  get_fields_if_exist(event.Data, "field", "clientKey")
			// what if we miss certs/key? do we have to panic or ignore it?
			if ok1 && ok2 && ok3 {
				host.Data["caCrt"] = caCrt.(string)
				host.Data["clientCrt"] = clientCrt.(string)
				host.Data["clientKey"] = clientKey.(string)
			} else{
				logrus.Error("Missing certs/key for clusterConnection for connection " +
				clusterConnection.(string))
				panic(errors.New("Missing certs/key for clusterConnection for connection " +
				clusterConnection.(string)))
			}
		}
	}
	return &instance, &host
}

func Is_instance_active(instance *model.Instance, host *model.Host) bool {
	if is_no_op(instance.Data) {
		return true
	}

	client := docker_client.Get_client(DEFAULT_VERSION)
	container := get_container(client, instance, false)
	return is_running(client, container)
}

func is_no_op(data map[string]interface{}) bool {
	b, ok := get_fields_if_exist(data, "containerNoOpEvent")
	if ok {
		return b.(bool)
	}
	return false
}

func get_container(client *client.Client, instance *model.Instance, by_agent bool) *types.Container {
	if instance == nil {
		return nil
	}

	// First look for UUID label directly
	args := filters.NewArgs()
	args.Add("label", fmt.Sprintf("%s=%s", UUID_LABEL, instance.UUID))
	options := types.ContainerListOptions{All:true, Filter: args}
	labeled_containers, err := client.ContainerList(context.Background(), options)
	if err == nil {
		return &labeled_containers[0]
	}

	// Nest look by UUID using fallback method
	options = types.ContainerListOptions{All:true}
	container_list, err := client.ContainerList(context.Background(), options)
	if err != nil {
		return nil
	}
	container := find_first(&container_list, func (c *types.Container) bool{
		if get_uuid(c) == instance.UUID{
			return true
		}
		return false
	})

	if container != nil {
		return container
	}
	if externalId := instance.ExternalId; externalId != "" {
		container = find_first(&container_list, func (c *types.Container) bool {
			return id_filter(externalId, c)
		})
	}

	if container != nil {
		return container
	}

	if agentId := instance.AgentId; by_agent{
		container = find_first(&container_list, func (c *types.Container) bool {
			return agent_id_filter(string(agentId), c)
		})
	}

	return container


}

func is_running(client *client.Client, container *types.Container) bool {
	if container == nil {
		return false
	}
	inspect, err := client.ContainerInspect(context.Background(), container.ID)
	if err == nil {
		return inspect.State.Running
	}
	return false
}

func get_uuid(container *types.Container) string {
	uuid, err := container.Labels[UUID_LABEL]
	if err {
		return uuid
	}

	names := container.Names
	if names == nil {
		return fmt.Sprintf("no-uuid-%s", container.ID)
	}

	if strings.HasPrefix(names[0], "/") {
		return names[0][1:]
	} else {
		return names[0]
	}
}

func find_first(containers *[]types.Container, f func(*types.Container) bool) *types.Container {
	for _, c := range *containers {
		if f(&c) {
			return &c
		}
	}
	return nil
}

func id_filter(id string, container *types.Container) bool {
	return container.ID == id
}

func agent_id_filter(id string, container *types.Container) bool {
	 container_id, ok := container.Labels["io.rancher.container.agent_id"]
	if ok {
		return container_id == id
	}
	return false
}

func Record_state(client *client.Client, instance *model.Instance, docker_id string) {
	if len(docker_id) > 0 {
		container := get_container(client, instance, false)
		if container != nil {
			docker_id = container.ID
		}
	}

	if len(docker_id) > 0 {
		return
	}

}

func Do_instance_activate(instance *model.Instance, host *model.Host, progress *progress.Progress) (error) {
	if is_no_op(instance.Data) {
		return nil
	}

	client := docker_client.Get_client(DEFAULT_VERSION)

	image_tag, err := get_image_tag(instance)
	if err != nil {
		logrus.Debug(err)
		return err
	}
	name := instance.UUID
	instance_name := instance.Name
	if len(instance_name) > 0 && len(instance_name) > 0 {
		if ok, _ := regexp.Match("^[a-zA-Z0-9][a-zA-Z0-9_.-]+$", []byte(instance_name)); ok {
			id := fmt.Sprintf("r-%s", name)
			_, err := client.ContainerInspect(context.Background(), id)
			if err != nil {
				name = id
			}
		}
	}

	var create_config = map[string]interface{} {
		"name" : name,
		"detach" : true,
	}

	var start_config = map[string]interface{}{
		"publish_all_ports" : false,
		"privileged" : is_true(instance, "privileged"),
		"read_only": is_true(instance,"readOnly"),
	}

	// These _setup_simple_config_fields calls should happen before all
	// other config because they stomp over config fields that other
	// setup methods might append to. Example: the environment field
	setup_simple_config_fields(create_config, instance,
		CREATE_CONFIG_FIELDS)

	setup_simple_config_fields(start_config, instance,
		START_CONFIG_FIELDS)

	add_label(create_config, map[string]string{UUID_LABEL: instance.UUID})

	if len(instance_name) > 0 {
		add_label(create_config, map[string]string{"io.rancher.container.name": instance_name})
	}

	setup_dns_search(start_config, instance)

	setup_logging(start_config, instance)

	setup_hostname(create_config, instance)

	setup_command(create_config, instance)

	setup_ports(create_config, instance, start_config)

	setup_volumes(create_config, instance, start_config, client)

	setup_links(start_config, instance)

	setup_networking(instance, host, create_config, start_config)

	flag_system_container(instance, create_config)

	setup_proxy(instance, create_config)

	setup_cattle_config_url(instance, create_config)

	create_config["host_config"] = create_host_config(start_config)

	setup_device_options(create_config["host_config"].(container.HostConfig), instance)

	container_id := get_container(client, instance, false).ID
	created := false
	if len(container_id) > 0 {
		container_id = create_container(client, create_config, image_tag, instance, name, progress)
		created = true
	}

	logrus.Info(fmt.Sprintf("Starting docker container [%s] docker id [%s] %v", name, container_id, start_config))

	start_err := client.ContainerStart(context.Background(), container_id, types.ContainerStartOptions{})

	if start_err != nil {
		if created {
			if err1 := remove_container(client, container_id); err1 != nil {
				logrus.Error(err1)
			}
		}
	}

	Record_state(client, instance, container_id)
	return nil
}

func create_container(client *client.Client, create_config map[string]interface{},
	image_tag string, instance *model.Instance, name string, progress *progress.Progress) string {
	logrus.Info(fmt.Sprintf("Creating docker container [%s] from config %v", name, create_config))
	// debug
	logrus.Info("debug")
	labels := create_config["labels"]
	if labels.(map[string]string)["io.rancher.container.pull_image"] == "always" {
		do_instance_pull(&model.Image_Params{
			Image: instance.Image,
			Tag: "",
			Mode: "all",
			Complete: false,
		}, progress)
	}
	delete(create_config, "name")
	command := ""
	if create_config["command"] != nil {
		command = create_config["command"].(string)
	}
	delete(create_config, "command")
	config := create_container_config(image_tag, command, create_config)
	host_config := create_config["host_config"].(container.HostConfig)


	if v_driver, ok := get_fields_if_exist(instance.Data, "field", "volumeDriver"); ok {
		host_config.VolumeDriver = v_driver.(string)
	}

	container_response, err := client.ContainerCreate(context.Background(), config, &host_config, nil, name)
	// if image doesn't exist
	if err != nil {
		if strings.Contains(err.Error(), config.Image) {
			pull_image(instance.Image, progress)
			container_response, err1 := client.ContainerCreate(context.Background(), config, &host_config, nil, name)
			if err1 != nil {
				logrus.Error(fmt.Sprintf("container id %s fail to start", container_response.ID))
				panic(err1)
			}
		} else {
			panic(err)
		}
	}

	return container_response.ID
}

func remove_container(client *client.Client, container_id string) error {
	err := client.ContainerRemove(context.Background(), container_id, types.ContainerRemoveOptions{})
	return err
}

func do_instance_pull(params *model.Image_Params, progress *progress.Progress) (types.ImageInspect, error) {
	client := docker_client.Get_client(DEFAULT_VERSION)

	image_json, ok := get_fields_if_exist(params.Image.Data, "dockerImage")
	if !ok {
		return types.ImageInspect{}, errors.New("field not exist")
	}
	var docker_image model.DockerImage
	mapstructure.Decode(image_json, &docker_image)
	existing, _, err := client.ImageInspectWithRaw(context.Background(), docker_image.ID, false)
	if err != nil {
		return types.ImageInspect{}, err
	}
	if params.Mode == "cached" {
		return existing, nil
	}
	if params.Complete {
		var err1 error
		_, err1 = client.ImageRemove(context.Background(), docker_image.ID, types.ImageRemoveOptions{})
		return types.ImageInspect{}, err1
	}

	image_pull(params, progress)

	if len(params.Tag) > 0 {
		image_info := parse_repo_tag(docker_image.FullName)
		repo_tag := fmt.Sprintf("%s:%s", image_info["repo"], image_info["tag"] + params.Tag)
		client.ImageTag(context.Background(), docker_image.ID, repo_tag)
	}

	inspect, _, err2 := client.ImageInspectWithRaw(context.Background(), docker_image.ID, false)
	return inspect, err2
}

func create_container_config(image_tag string, command string, create_config map[string]interface{}) *container.Config {
	create_config["cmd"] = []string{command}
	create_config["image"] = image_tag
	var config container.Config
	err := mapstructure.Decode(create_config, &config)
	if err != nil {
		panic(err)
	}
	return &config
}

func get_image_tag(instance *model.Instance) (string, error){
	var docker_image model.DockerImage
	mapstructure.Decode(instance.Image.Data["dockerImage"], &docker_image)
	image_name := docker_image.FullName
	if image_name == "" {
		return "", errors.New("Can not start container with no image")
	}
	return image_name, nil
}

func is_true(instance *model.Instance, field string) bool {
	_, ok := get_fields_if_exist(instance.Data, field)
	return ok
}

func setup_simple_config_fields(config map[string]interface{}, instance *model.Instance, fields []model.Tuple){
	for _, tuple := range fields {
		src := tuple.Src
		dest := tuple.Dest
		src_obj, ok := get_fields_if_exist(instance.Data, "field", src)
		if !ok {
			break
		}
		config[dest] = unwrap(&src_obj)
	}
}

func setup_dns_search(start_config map[string]interface{}, instance *model.Instance){
	container_id := instance.SystemContainer
	if len(container_id) == 0 {
		return
	}
	// if only rancher search is specified,
	// prepend search with params read from the system
	all_rancher := true
	dns_search, ok2 := start_config["dns_search"].([]string)
	if ok2 {
		if dns_search == nil || len(dns_search) == 0 {
			return
		}
		for _, search := range dns_search {
			if strings.HasSuffix(search, "rancher.internal") {
				continue
			}
			all_rancher = false
			break
		}
	} else {
		return
	}

	if !all_rancher {
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
			for i, _ := range s {
				search := s[len(s)-i-1]
				if !search_in_list(s, search) {
					dns_search = append(dns_search, search)
				}
			}
			start_config["dns_search"] = dns_search
		}
	}

}

func setup_logging(start_config map[string]interface{}, instance *model.Instance){
	log_config, ok := start_config["log_config"]
	if !ok {
		return
	}
	driver, ok := log_config.(map[string]interface{})["driver"].(string)

	if ok {
		delete(start_config["log_config"].(map[string]interface{}), "driver")
		start_config["log_config"].(map[string]interface{})["type"] = driver
	}

	for _, value := range []string{"type", "config"} {
		bad := true
		obj, ok := start_config["log_config"].(map[string]interface{})[value]
		if ok && obj != nil {
			bad = false
			start_config["log_config"].(map[string]interface{})[value] = unwrap(&obj)
		}
		if _, ok1 := start_config["log_config"]; bad && ok1 {
			delete(start_config, "log_config")
		}
	}


}

func setup_hostname(create_config map[string]interface{}, instance *model.Instance){
	name := instance.Hostname
	if len(name) > 0 {
		create_config["hostname"] = name
	}
}

func setup_command(create_config map[string]interface{}, instance *model.Instance){
	command, ok := get_fields_if_exist(instance.Data, "field", "command")
	if !ok {
		return
	}
	switch command.(type) {
	case string : setup_legacy_command(create_config, instance, command.(string))
	default:
		if command != nil {
			create_config["command"] = command
		}
	}
}

func setup_ports(create_config map[string]interface{}, instance *model.Instance,
	start_config map[string]interface{}){
	ports := []model.Port{}
	bindings := make(map[string]interface{})
	if instance.Ports != nil && len(instance.Ports) > 0 {
		for _, port := range instance.Ports {
			ports = append(ports, model.Port{PrivatePort: port.PrivatePort, Protocol: port.Protocol})
			if port.PrivatePort != 0 {
				bind := fmt.Sprintf("%s/%s", port.PrivatePort, port.Protocol)
				bind_addr := ""
				if bindAddress, ok :=  get_fields_if_exist(port.Data, "fields", "bindAddress"); ok {
					bind_addr = bindAddress.(string)
				}
				host_bind := model.Host_Bind{Bind_addr: bind_addr, PublicPort: port.PublicPort}
				if _, ok := bindings[bind]; !ok {
					bindings[bind] = []model.Host_Bind{host_bind}
				} else {
					bindings[bind] = append(bindings[bind].([]model.Host_Bind), host_bind)
				}
			}

		}
	}

	if len(ports) > 0 {
		create_config["port"] = ports
	}

	if len(bindings) > 0 {
		start_config["port_bindings"] = bindings
	}

}

func setup_volumes(create_config map[string]interface{}, instance *model.Instance,
	start_config map[string]interface{}, client *client.Client){
	if volumes, ok := get_fields_if_exist(instance.Data, "field", "dataVolumes"); ok {
		volumes := volumes.([]string)
		volumes_map := make(map[string]interface{})
		binds_map := make(map[string]interface{})
		if len(volumes) > 0 {
			for _, volume := range volumes {
				parts := strings.SplitAfterN(volume, ":", 3)
				if len(parts) == 1 {
					volumes_map[parts[0]] = make(map[string]interface{})
				} else {
					mode := ""
					if len(parts) == 3 {
						mode = parts[2]
					} else {
						mode = "rw"
					}
					bind := struct{
						Binds string
						Mode string
					} {
						parts[1],
						mode,
					}
					binds_map[parts[0]] = bind
				}
			}
			create_config["volumes"] = volumes_map
			start_config["binds"] = binds_map
		}
	}

	containers := []string{}
	if vfs_list := instance.DataVolumesFromContainers; vfs_list != nil {
		for _, vfs := range vfs_list {
			var in model.Instance
			mapstructure.Decode(vfs, &in)
			container := get_container(client, &in, false)
			if container != nil {
				containers = append(containers,container.ID)
			}
		}
		if containers != nil && len(containers) > 0 {
			start_config["volumes_from"] = containers
		}
	}

	if v_mounts := instance.DataVolumesFromContainers; len(v_mounts) > 0 {
		for v_mount := range v_mounts {
			var volume model.Volume
			err := mapstructure.Decode(v_mount, &volume)
			if err != nil {
				if !is_volume_active(volume) {
					do_volume_activate(volume)
				}
			} else {
				panic(err)
			}
		}
	}
}

func setup_links(start_config map[string]interface{}, instance *model.Instance){
	links := make(map[string]interface{})

	if instance.InstanceLinks == nil {
		return
	}

	for _, link := range instance.InstanceLinks {
		if link.TargetInstanceId != "" {
			links[link.TargetInstance.UUID] = link.LinkName
		}
	}
	start_config["links"] = links

}

func setup_networking(instance *model.Instance, host *model.Host,
	create_config map[string]interface{}, start_config map[string]interface{}){
	client := docker_client.Get_client(DEFAULT_VERSION)
	ports_supported, hostname_supported := setup_network_mode(instance, client, create_config, start_config)
	setup_mac_and_ip(instance, create_config, ports_supported, hostname_supported)
	setup_ports_network(instance, create_config, start_config, ports_supported)
	setup_links_network(instance, create_config, start_config)
	setup_ipsec(instance, host, create_config, start_config)
	setup_dns(instance)
}

func flag_system_container(instance *model.Instance, create_config map[string]interface{}){
	if len(instance.SystemContainer) > 0  {
		add_label(create_config, map[string]string{"io.rancher.container.system": instance.SystemContainer})
	}
}

func setup_proxy(instance *model.Instance, create_config map[string]interface{}){
	if len(instance.SystemContainer) > 0 {
		if !has_key(create_config, "environment") {
			create_config["environment"] = map[string]interface{}{}
		}
		for _, i := range []string{"http_proxy", "https_proxy", "NO_PROXY"} {
			create_config["enviroment"].(map[string]interface{})[i] = os.Getenv(i)
		}
	}
}

func setup_cattle_config_url(instance *model.Instance, create_config map[string]interface{}){
	if instance.AgentId >= 0 && has_label(instance) {
		return
	}

	if has_key(create_config, "labels") {
		create_config["labels"] = make(map[string]string)
	}
	create_config["labels"].(map[string]string)["io.rancher.container.agent_id"] = strconv.Itoa(instance.AgentId)

	url := config_url()

	if len(url) > 0 {
		parsed, err := urls.Parse(url)

		if err != nil {
			logrus.Error(err)
			panic(err)
		} else{
			if parsed.Host == "localhost" {
				port := api_proxy_listen_port()
				add_to_env(create_config, map[string]string{
					"CATTLE_AGENT_INSTANCE": "true",
					"CATTLE_CONFIG_URL_SCHEME": parsed.Scheme,
					"CATTLE_CONFIG_URL_PATH": parsed.Path,
					"CATTLE_CONFIG_URL_PORT": string(port),
				})
			} else {
				add_to_env(create_config, map[string]string{
					"CATTLE_CONFIG_URL": url,
					"CATTLE_URL": url,
				})
			}
		}
	}
}

func setup_device_options(config container.HostConfig, instance *model.Instance){
	option_configs := []model.Option_Config{
		model.Option_Config{
			Key: "readIops",
			Dev_List: []map[string]string{},
			Docker_Field: "BlkioDeviceReadIOps",
			Field: "Rate",
		},
		model.Option_Config{
			Key: "writeIops",
			Dev_List: []map[string]string{},
			Docker_Field: "BlkioDeviceWriteIOps",
			Field: "Rate",
		},
		model.Option_Config{
			Key: "readBps",
			Dev_List: []map[string]string{},
			Docker_Field: "BlkioDeviceReadBps",
			Field: "Rate",
		},
		model.Option_Config{
			Key: "writeBps",
			Dev_List: []map[string]string{},
			Docker_Field: "BlkioDeviceWriteBps",
			Field: "Rate",
		},
		model.Option_Config{
			Key: "weight",
			Dev_List: []map[string]string{},
			Docker_Field: "BlkioWeightDevice",
			Field: "Weight",
		},
	}

	if device_options, ok := get_fields_if_exist(instance.Data, "field", "blkioDeviceOptions"); !ok {
		return
	} else {
		device_options := device_options.(map[string]map[string]string)
		for dev, options := range device_options{
			if dev == "DEFAULT_DICK" {
				//dev = host_info.Get_default_disk()
				if len(dev) == 0 {
					logrus.Warn(fmt.Sprintf("Couldn't find default device. Not setting device options: %s", options))
					continue
				}
			}
			for _, o_c := range option_configs {
				key, dev_list, field := o_c.Key, o_c.Dev_List, o_c.Field
				if has_key(options, key) && len(options[key]) > 0 {
					value := options[key]
					dev_list = append(dev_list, map[string]string{"Path": dev, field: value})
				}
			}
		}
		/*
		for _, o_c := range option_configs {
			dev_list, docker_field := o_c.Dev_List, o_c.Docker_Field
			if len(dev_list) >0 {
				config.D = dev_list
			}
		}
		*/

	}
}

func setup_legacy_command(create_config map[string]interface{}, instance *model.Instance, command string){
	// This can be removed shortly once cattle removes
	// commandArgs
	if len(command) > 0 || len(strings.TrimSpace(command)) == 0 {
		return
	}

	command_args := []string{}
	if value := instance.Command_args; value != nil {
		command_args = value
	}
	commands := []string{}
	if command_args != nil && len(command_args) > 0 {
		commands = append(commands, command)
		for _, value := range command_args {
			commands = append(commands, value)
		}
	}

	if len(commands) > 0 {
		create_config["command"] = commands
	}
}

func create_host_config(start_config map[string]interface{}) container.HostConfig {
	var host_config container.HostConfig
	err := mapstructure.Decode(start_config, &host_config)
	if err == nil {
		return host_config
	} else {
		panic(err)
	}
}



