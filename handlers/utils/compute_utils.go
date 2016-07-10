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
	"../../model"
	"../progress"
	"github.com/docker/engine-api/types"
	"../docker_client"
	"github.com/rancher/go-machine-service/events"
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
	ihm := data["instanceHostMap"].(model.InstanceHostMap)

	var instance model.Instance
	instance = mapstructure.Decode(ihm.Instance, &instance)
	var host model.Host
	host = mapstructure.Decode(ihm.Host, &host)

	var clusterConnection string
	clusterConnection, ok := get_fields_if_exist(data, "field", "clusterConnection")
	if ok {
		logrus.Debugf("clusterConnection = %s", clusterConnection)
		host.Data["clusterConnection"] = clusterConnection
		if strings.HasPrefix(clusterConnection, "http") {
			var caCrt, clientCrt, clientKey string
			caCrt, ok1 := get_fields_if_exist(event.Data, "field", "caCrt")
			clientCrt, ok2 :=  get_fields_if_exist(event.Data, "field", "clientCrt")
			clientKey, ok3 :=  get_fields_if_exist(event.Data, "field", "clientKey")
			// what if we miss certs/key? do we have to panic or ignore it?
			if ok1 == nil && ok2 == nil && ok3 == nil {
				host.Data["caCrt"] = caCrt
				host.Data["clientCrt"] = clientCrt
				host.Data["clientKey"] = clientKey
			} else{
				logrus.Error("Missing certs/key for clusterConnection for connection " +
				clusterConnection)
				panic(errors.New("Missing certs/key for clusterConnection for connection " +
				clusterConnection))
			}
		}
	}
	return &instance, &host
}

func Is_instance_active(instance *model.Instance, host model.Host) bool {
	if is_no_op(instance) {
		return true
	}

	client := docker_client.Get_client(DEFAULT_VERSION)
	container := get_container(client, instance, false)
	return is_running(client, container)
}

func is_no_op(instance *model.Instance) bool {
	b, ok := get_fields_if_exist(instance.Data, "containerNoOpEvent")
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
	options := types.ContainerListOptions{All:true, Filter: {"label": fmt.Sprintf("%s=%s", UUID_LABEL, instance.UUID)}}
	labeled_containers, err := client.ContainerList(context.Background(), options)
	if err == nil {
		return labeled_containers[0]
	}

	// Nest look by UUID using fallback method
	options = types.ContainerListOptions{All:true}
	container_list, err := client.ContainerList(context.Background(), options)
	if err != nil {
		return nil
	}
	container := find_first(container_list, func (c types.Container) bool{
		if strings.Compare(get_uuid(c), instance.UUID){
			return true
		}
		return false
	})

	if container != nil {
		return container
	}
	if externalId := instance.ExternalId; externalId != "" {
		container = find_first(container_list, func (id string) bool{
			return id_filter(externalId, id)
		})
	}

	if container != nil {
		return container
	}

	if agentId := instance.AgentId; agentId != "" && by_agent{
		container = find_first(container_list, func (id string) bool{
			return agent_id_filter(agentId, id)
		})
	}

	return &container


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

func find_first(containers []types.Container, f func(string) bool) *types.Container {
	for _, c := range containers {
		if f(c.ID) {
			return &c
		}
	}
	return nil
}

func id_filter(id string, container *types.Container) bool {
	container_id := container.ID
	return strings.Compare(container_id, id)
}

func agent_id_filter(id string, container *types.Container) bool {
	 container_id, ok := container.Labels["io.rancher.container.agent_id"]
	if ok {
		return strings.Compare(container_id, id)
	}
	return false
}

func Record_state(client *client.Client, instance *model.Instance, docker_id string) {
	if docker_id == nil {
		container := get_container(client, instance, false)
		if container != nil {
			docker_id = container.ID
		}
	}

	if docker_id == nil {
		return nil
	}

}

func Do_instance_activate(instance *model.Instance, host *model.Host, progress interface{}){
	if is_no_op(instance) {
		return
	}

	client := docker_client.Get_client(DEFAULT_VERSION)

	image_tag, err := get_image_tag(instance)
	if err != nil {
		logrus.Debug(err)
		panic(err)
	}
	name := instance.UUID
	instance_name := instance.Name
	if instance_name != nil && len(instance_name) > 0 && regexp.Match("^[a-zA-Z0-9][a-zA-Z0-9_.-]+$", instance_name){
		id := fmt.Sprintf("r-%s", name)
		_, err := client.ContainerInspect(context.Background(), id)
		if err != nil {
			name = id
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

	add_label(create_config, map[string]string{UUID_LABEL: instance["uuid"].(string)})

	if instance_name != nil && len(instance_name) > 0 {
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

	setup_device_options(create_config["host_config"], instance)

	container := get_container(client, instance, false)
	created := false
	if container == nil {
		container = create_container(client, create_config, image_tag, instance, name, progress)
		created = true
	}

	container_id := container.ID
	logrus.Info(fmt.Sprintf("Starting docker container [%s] docker id [%s] %s", name, container_id, start_config))

	start_err := client.ContainerStart(context.Background(), container_id, nil)

	if start_err!= nil {
		if created {
			if err1 := remove_container(client, container); err1 != nil {
				logrus.Error(err1)
			}
		}
	}

	Record_state(client, instance, container_id)

}

func create_container(client *client.Client, create_config *map[string]interface{},
	image_tag string, instance *model.Instance, name string, progress *progress.Progress){
	logrus.Info(fmt.Sprintf("Creating docker container [%s] from config %s", name, create_config))

	labels := create_config["labels"]
	if labels.(map[string]string)["io.rancher.container.pull_image"] == "always" {
		do_instance_pull(&model.Image_Params{
			Image: instance.Image,
			Tag: nil,
			Mode: "all",
			Complete: false,
		}, progress)
	}
	delete(create_config, "name")
	command := ""
	command = create_config["command"]
	delete(create_config, "command")
	config := create_container_config(image_tag, command, create_config)
	host_config := create_config["host_config"].(model.Host_Config)

	if v_driver, ok := get_fields_if_exist(instance, "field", "volumeDriver"); ok {
		config["VolumeDriver"] = v_driver.(string)
	}
	container, err := client.ContainerCreate(context.Background(), config, host_config, nil, name)
	// if image doesn't exist
	if err != nil {
		if strings.Contains(err.Error(), config.Image) {
			pull_image(instance.Image, progress)
			container = client.ContainerCreate(context.Background(), config, host_config, nil, name)
		} else {
			panic(err)
		}
	}
	return container
}

func remove_container(client *client.Client, container *types.Container) error {
	err := client.ContainerRemove(context.Background(), container.ID, nil)
	return err
}

func do_instance_pull(params *model.Image_Params, progress *progress.Progress) (types.ImageInspect, error) {
	client := docker_client.Get_client(DEFAULT_VERSION)

	image_json, ok := get_fields_if_exist(params.Image.Data, "dockerImage")
	if ok != nil {
		return nil
	}
	var docker_image model.DockerImage
	mapstructure.Decode(image_json, &docker_image)
	existing, _, err := client.ImageInspectWithRaw(context.Background(), docker_image.ID, false)
	if err != nil {
		return nil, err
	}
	if params.Mode == "cached" && existing == nil {
		return existing, nil
	}
	if params.Complete {
		var err error
		if existing != nil {
			err = client.ImageRemove(context.Background(), docker_image.ID, false)
		}
		return nil, err
	}

	image_pull(params.Image, progress)

	if params.Tag != nil {
		image_info := parse_repo_tag(docker_image.FullName)
		repo_tag := fmt.Sprintf("%s:%s", image_info["repo"], image_info["tag"] + params.Tag)
		client.ImageTag(context.Background(), docker_image, repo_tag)
	}

	return client.ImageInspectWithRaw(context.Background(), docker_image.ID, false)
}

func create_container_config(image_tag string, command string, create_config string) model.Config {
	create_config["cmd"] = command
	create_config["image"] = image_tag
	var config model.Config
	err := mapstructure.Decode(create_config, &config)
	if err != nil {
		panic(err)
	}
	return config
}

func get_image_tag(instance *model.Instance) (string, error){
	image_name := instance.Image["data"]["dockerImage"].(model.DockerImage).FullName
	if image_name != "" {
		return nil, errors.New("Can not start container with no image")
	}
	return image_name, nil
}

func is_true(instance *model.Instance, field string) bool {
	_, ok := get_fields_if_exist(instance, "field", field)
	if ok != nil {
		return false
	}
	return true
}

func setup_simple_config_fields(config *map[string]interface{}, instance *model.Instance, fields []model.Tuple){
	for _, tuple := range fields {
		src := tuple.Src
		dest := tuple.Dest
		src_obj, ok := get_fields_if_exist(instance.Data, "field", src)
		if ok != nil {
			return nil
		}
		config[dest] = unwrap[src_obj]
	}
}

func setup_dns_search(start_config *map[string]interface{}, instance *model.Instance){
	b, ok1 := instance["systemContainer"]
	if ok1 && b {
		return nil
	}
	// if only rancher search is specified,
	// prepend search with params read from the system
	all_rancher := true
	dns_search, ok2 := start_config["dns_search"].([]string)
	if ok2 {
		if dns_search == nil || len(dns_search) == 0 {
			return nil
		}
		for _, search := range dns_search {
			if strings.HasSuffix(search, "rancher.internal") {
				continue
			}
			all_rancher = false
			break
		}
	} else {
		return nil
	}

	if !all_rancher {
		return nil
	}

	// read host's resolv.conf
	file, err := os.Open("/etc/resolv.conf")
	if err != nil {
		logrus.Error(err)
		return nil
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
					append([]string{search}, dns_search)
				}
			}
			start_config["dns_search"] = dns_search
		}
	}

}

func setup_logging(start_config *map[string]interface{}, instance *model.Instance){
	log_config, ok := start_config["log_config"]
	if !ok {
		return nil
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
			start_config["log_config"].(map[string]interface{})[value] = unwrap(obj)
		}
		if _, ok1 := start_config["log_config"]; bad && ok1 {
			delete(start_config, "log_config")
		}
	}


}

func setup_hostname(create_config *map[string]interface{}, instance *model.Instance){
	name, ok := instance.Hostname
	if ok {
		create_config["hostname"] = name
	}
}

func setup_command(create_config *map[string]interface{}, instance *model.Instance){
	command := ""
	command, ok := get_fields_if_exist(instance.Data, "field", "command")
	if !ok {
		return nil
	}
	switch command.(type) {
	case string : setup_legacy_command(create_config, instance, command)
	default:
		if command != nil {
			create_config["command"] = command
		}
	}
}

func setup_ports(create_config *map[string]interface{}, instance *model.Instance,
	start_config *map[string]interface{}){
	ports := []model.Port{}
	bindings := make(map[string]interface{})
	if instance.Ports != nil && len(instance.Ports) > 0 {
		for port := range instance.Ports {
			append(ports, model.Port{PrivatePort: port["privatePort"], Protocol: port["protocol"]})
			if public_port, ok := port["publicPort"]; ok && public_port != nil {
				bind := fmt.Sprintf("%s/%s", port["privatePort"], port["protocol"])
				bind_addr := ""
				if bindAddress, ok := port["data"].(map[string]interface{})["fields"].
				(map[string]interface{})["bindAddress"]; ok && bindAddress != nil {
					bind_addr = bindAddress
				}
				host_bind := model.Host_Bind{Bind_addr: bind_addr, PublicPort: public_port}
				if _, ok := bindings[bind]; !ok {
					bindings[bind] = []model.Host_Bind{host_bind}
				} else {
					append(bindings[bind], host_bind)
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

func setup_volumes(create_config *map[string]interface{}, instance *model.Instance,
	start_config *map[string]interface{}, client *client.Client){
	if volumes, ok := get_fields_if_exist(instance, "field", "dataVolumes"); ok && volumes != nil {
		volumes = volumes.([]string)
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
			container := get_container(client, vfs, false)
			if container != nil {
				append(containers,container.ID)
			}
		}
		if containers != nil && len(containers) > 0 {
			start_config["volumes_from"] = containers
		}
	}

	if v_mounts, ok := instance.DataVolumesFromContainers; ok {
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

func setup_links(start_config *map[string]interface{}, instance *model.Instance){
	links := make(map[string]interface{})

	if instance.InstanceLinks == nil {
		return nil
	}

	for _, link := range instance.InstanceLinks {
		if link.TargetInstanceId != "" {
			links[link.TargetInstance.UUID] = link.LinkName
		}
	}
	start_config["links"] = links

}

func setup_networking(instance *model.Instance, host *map[string]interface{},
	create_config *map[string]interface{}, start_config *map[string]interface{}){
	client := docker_client.Get_client(DEFAULT_VERSION)
	ports_supported, hostname_supported := setup_network_mode(instance, client, &create_config, &start_config)
	setup_mac_and_ip(instance, create_config, ports_supported, hostname_supported)
	setup_ports_network(instance, create_config, start_config, ports_supported)
	setup_links_network(instance, create_config, start_config)
	setup_ipsec(instance, host, create_config, start_config)
	setup_dns(instance)
}

func flag_system_container(instance *model.Instance, create_config *map[string]interface{}){
	if instance.SystemContainer != nil {
		add_label(create_config, map[string]string{"io.rancher.container.system": instance.SystemContainer})
	}
}

func setup_proxy(instance *model.Instance, create_config *map[string]interface{}){
	if instance.SystemContainer != nil {
		if !has_key(create_config, "environment") {
			create_config["environment"] = make(map[string]interface{})
		}
		for _, i := range []string{"http_proxy", "https_proxy", "NO_PROXY"} {
			create_config["enviroment"][i] = os.Getenv(i)
		}
	}
}

func setup_cattle_config_url(instance *model.Instance, create_config *map[string]interface{}){
	if instance.AgentId == nil && has_label(instance) {
		return nil
	}

	if has_key(create_config, "labels") {
		create_config["labels"] = make(map[string]string)
	}
	create_config["labels"]["io.rancher.container.agent_id"] = strconv.Itoa(instance.AgentId)

	url := config_url()

	if url != nil {
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
					"CATTLE_CONFIG_URL_PORT": port,
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

func setup_device_options(config *map[string]interface{}, instance *model.Instance){
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
		return nil
	} else {
		device_options = device_options.(map[string]map[string]string)
		for dev, options := range device_options{
			if dev == "DEFAULT_DICK" {
				//dev = host_info.Get_default_disk()
				if dev == nil {
					logrus.Warn(fmt.Sprintf("Couldn't find default device. Not setting".
					"device options: %s", options))
					continue
				}
			}
			for _, o_c := range option_configs {
				key, dev_list, field := o_c.Key, o_c.Dev_List, o_c.Field
				if has_key(options, key) && options[key] != nil {
					value := options[key]
					append(dev_list, map[string]string{"Path": dev, field: value})
				}
			}
		}
		for _, o_c := range option_configs {
			dev_list, docker_field := o_c.Dev_List, o_c.Docker_Field
			if len(dev_list) >0 {
				config[docker_field] = dev_list
			}
		}

	}
}

func setup_legacy_command(create_config *map[string]interface{}, instance *model.Instance, command string){
	// This can be removed shortly once cattle removes
	// commandArgs
	if command == nil || len(strings.TrimSpace(command)) == 0 {
		return nil
	}

	command_args := []string{}
	if value := instance.Command_args; value != nil {
		command_args = value
	}

	if command_args != nil && len(command_args) > 0 {
		command = []string{command}
		append(command, command_args)
	}

	if command != nil {
		create_config["command"] = command
	}
}

func create_host_config(start_config *map[string]interface{}) model.Host_Config {
	var host_config model.Host_Config
	err := mapstructure.Decode(start_config, &host_config)
	if err == nil {
		return host_config
	} else {
		panic(err)
	}
}



