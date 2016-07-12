package utils

import (
	"github.com/docker/engine-api/client"
	"fmt"
	"strings"
	"github.com/mitchellh/mapstructure"
	"regexp"
	"strconv"
	"github.com/rancher/agent/model"
	"github.com/Sirupsen/logrus"
)

func setupMacAndIp(instance *model.Instance, create_config map[string]interface{}, set_mac bool, set_hostname bool) {
	/*
	Configures the mac address and primary ip address for the the supplied
	container. The mac_address is configured directly as part of the native
	docker API. The primary IP address is set as an environment variable on the
	container. Another Rancher micro-service will detect this environment
	variable when the container is started and inject the IP into the
	container.

	Note: while an instance can technically have more than one nic based on the
	resource schema, this implementation assumes a single nic for the purpose
	of configuring the mac address and IP.
	*/
	mac_address := ""
	device_number := -1
	for _, nic := range instance.Nics {
		if device_number == -1 {
			mac_address = nic.MacAddress
			device_number = nic.DeviceNumber
		} else if device_number > nic.DeviceNumber{
			mac_address = nic.MacAddress
			device_number = nic.DeviceNumber
		}
	}

	if set_mac {
		create_config["mac_address"] = mac_address
	}

	if !set_hostname {
		delete(create_config, "hostname")
	}

	if instance.Nics != nil && len(instance.Nics) > 0 && instance.Nics[0].IPAddresses != nil {
		// Assume one nic
		nic := instance.Nics[0]
		logrus.Info("nic info %v", nic)
		ip_address := ""
		for _, ip := range nic.IPAddresses {
			logrus.Info("ip info %v", ip)
			if ip.Role == "primary" {
				ip_address = fmt.Sprintf("%s/%s", ip.Address, strconv.Itoa(ip.Subnet.CidrSize))
				break
			}
		}
		logrus.Info("ip info %s", ip_address)
		if ip_address != "" {
			addLabel(create_config, map[string]string{"io.rancher.container.ip": ip_address})
		}
	}
}

func setupNetworkMode(instance *model.Instance, client *client.Client,
	create_config map[string]interface{}, start_config map[string]interface{}) (bool, bool) {
	/*
	Based on the network configuration we choose the network mode to set in
    Docker.  We only really look for none, host, or container.  For all
    all other configurations we assume bridge mode
	 */
	ports_supported := true
	hostname_supported := true
	if len(instance.Nics) >0 {
		kind := instance.Nics[0].Network.Kind
		if kind == "dockermodel.model.Host" {
			ports_supported = false
			hostname_supported = false
			start_config["network_mode"] = "host"
			delete(start_config, "link")
		} else if kind == "dockerNone"{
			ports_supported = false
			create_config["network_mode"] = "none"
			delete(start_config, "link")
		} else if kind == "dockerContainer" {
			ports_supported = false
			hostname_supported = false
			id := instance.NetworkContainer["uuid"]
			var in model.Instance
			mapstructure.Decode(instance.NetworkContainer, &in)
			other := getContainer(client, &in, false)
			if other != nil {
				id = other.ID
			}
			start_config["network_mode"] = fmt.Sprintf("container:%s", id)
			delete(start_config, "link")
		}
	}
	return ports_supported, hostname_supported

}

func setupPortsNetwork(instance *model.Instance, create_config map[string]interface{},
	start_config map[string]interface{}, ports_supported bool){
	/*
	Docker 1.9+ does not allow you to pass port info for networks that don't
    support ports (net, none, container:x)
	 */
	if !ports_supported {
		start_config["publish_all_ports"] = false
		delete(create_config, "ports")
		delete(start_config, "port_bindings")
	}
}

func setupIpsec(instance *model.Instance, host *model.Host, create_config map[string]interface{},
	start_config map[string]interface{}){
	/*
	If the supplied instance is a network agent, configures the ports needed
    to achieve multi-host networking.
	 */
	network_agent := false
	if instance.SystemContainer == "" || instance.SystemContainer == "NetworkAgent" {
		network_agent = true
	}
	if !network_agent || !hasService(instance, "ipsecTunnelService") {
		return
	}
	host_id := strconv.Itoa(host.ID)
	if info, ok := instance.Data["ipsec"].(map[string]interface{})[host_id].(
		map[string]interface{}); ok {
		nat := info["nat"].(string)
		isakmp := info["isakmp"].(string)
		ports := getOrCreatePortList(create_config, "ports")
		binding := getOrCreateBindingMap(start_config, "port_bindings")

		// private port or public ?
		ports = append(ports, model.Port{PrivatePort: 500, Protocol: "udp"}, model.Port{PrivatePort: 4500, Protocol: "udp"})
		binding["500/udp"] = []string{"0.0.0.0", isakmp}
		binding["4500/udp"] = []string{"0.0.0.0", nat}
	}
}

func setupDns(instance *model.Instance){
	if !hasService(instance, "dnsService") || instance.Kind == "virtualMachine" {
		return
	}
	ip_address, mac_address, subnet := findIpAndMac(instance)

	if ip_address == "" || mac_address == "" {
		return
	}

	parts := strings.Split(ip_address, ".")
	if len(parts) != 4 {
		return
	}
	part2, _ := strconv.Atoi(parts[2])
	part3, _ := strconv.Atoi(parts[3])
	mark := strconv.Itoa(part2 * 1000 + part3)

	//TODO implement check_output function

	check_output([]string{"iptables", "-w", "-t", "nat", "-A", "CATTLE_PREROUTING",
                      "!", "-s", subnet, "-d", "169.254.169.250", "-m", "mac",
                      "--mac-source", mac_address, "-j", "MARK", "--set-mark",
                      mark})
        check_output([]string{"iptables", "-w", "-t", "nat", "-A", "CATTLE_POSTROUTING",
		"!", "-s", subnet, "-d", "169.254.169.250", "-m", "mark", "--mark", mark,
		"-j", "SNAT", "--to", ip_address})


}

func setupLinksNetwork(instance *model.Instance, create_config map[string]interface{},
	start_config map[string]interface{}){
	/*
	Sets up a container's config for rancher-managed links by removing the
    docker native link configuration and emulating links through environment
    variables.

    Note that a non-rancher container (one created and started outside the
    rancher API) container will not have its link configuration manipulated.
    This is because on a container restart, we would not be able to properly
    rebuild the link configuration because it depends on manipulating the
    create_config.
	 */
	if !hasService(instance, "linkService") || isNonrancherContainer(instance){
		return
	}

	if hasKey(start_config, "links") {
		delete(start_config, "links")
	}
	result := make(map[string]string)
	if instance.InstanceLinks != nil {
		for _, link := range instance.InstanceLinks {
			link_name := link.LinkName
			addLinkEnv(link_name, link, result, "")
			copyLinkEnv(link_name, link, result)
			if names, ok := link.Data["field"].(map[string]interface{})["instanceName"].([]string); ok {
				for _, name := range names {
					addLinkEnv(name, link, result, link_name)
					copyLinkEnv(name, link, result)
					// This does assume the format {env}_{name}
					parts := strings.SplitAfterN(name, "_", 1)
					if len(parts) == 1 {
						continue
					}
					addLinkEnv(name, link, result, link_name)
					copyLinkEnv(name, link, result)
				}

			}
		}
		if len(result) >0 {
			addToEnv(create_config, result)
		}
	}

}

func hasService(instance *model.Instance, kind string) bool {
	if instance.Nics != nil && len(instance.Nics) > 0 {
		for _, nic := range instance.Nics {
			if nic.DeviceNumber != 0 {
				continue
			}
			if nic.Network.NetworkServices != nil && len(nic.Network.NetworkServices) > 0 {
				for _, service := range nic.Network.NetworkServices {
					if service.Kind == kind {
						return true
					}
				}
			}

		}
	}
	return false
}

func addLinkEnv(name string, link model.Link, result map[string]string, in_ip string){
	result[strings.ToUpper(fmt.Sprintf("%s_NAME", toEnvName(name)))] = fmt.Sprintf("/cattle/%s", name)

	if ports, ok := link.Data["field"].(map[string]interface{})["link"]; !ok {
		return
	} else {
		for _, value := range ports.([]interface{}) {
			var port model.Port
			err := mapstructure.Decode(value, &port)
			if err != nil {
				panic(err)
			}
			protocol := port.Protocol
			ip := strings.ToLower(name)
			if in_ip != "" {
				ip = in_ip
			}
			// different with python agent
			dst := port.PublicPort
			src := port.PrivatePort

			full_port := fmt.Sprintf("%s://%s:%s", protocol, ip, dst)
			data := make(map[string]string)
			data["NAME"] = fmt.Sprintf("/cattle/%s", name)
			data["PORT"] = full_port
			data[fmt.Sprintf("PORT_%s_%s", src, protocol)] = full_port
			data[fmt.Sprintf("PORT_%s_%s_ADDR", src, protocol)] = ip
			data[fmt.Sprintf("PORT_%s_%s_PORT", src, protocol)] = string(dst)
			data[fmt.Sprintf("PORT_%s_%s_PROTO", src, protocol)] = protocol

			for key, value := range data {
				result[strings.ToUpper(fmt.Sprintf("%s_%s", toEnvName(name), key))] = value
			}
		}
	}
}

func copyLinkEnv(name string, link model.Link, result map[string]string){
	targetInstance := link.TargetInstance;
	if envs, ok := getFieldsIfExist(targetInstance.Data, "dockerInspect", "Config", "Env"); ok {
		ignores := make(map[string]bool)
		envs := envs.([]string)
		for _, env := range envs {
			parts := strings.SplitAfterN(env, "=", 1)
			if len(parts) == 1 {
				continue
			}
			if strings.HasPrefix(parts[1], "/cattle/") {
				env_name := toEnvName(parts[1][len("/cattle/"):])
				ignores[env_name + "_NAME"] = true
				ignores[env_name + "_PORT"] = true
				ignores[env_name + "_ENV"] = true
			}
		}

		for _, env := range envs {
			should_ingnore := false
			for ignore, _ := range ignores {
				if strings.HasPrefix(env, ignore) {
					should_ingnore = true
					break
				}
			}
			if should_ingnore {
				continue
			}
			parts := strings.SplitAfterN(env, "=", 1)
			if len(parts) == 1 {
				continue
			}
			key, value := parts[0], parts[1]
			if key == "HOME" || key == "PATH" {
				continue
			}
			result[fmt.Sprintf("%s_ENV_%s", toEnvName(name), key)] = value
		}
	}
}

func toEnvName(name string) string {
	r, err := regexp.Compile("[^a-zA-Z0-9_]")
	if err != nil {
		panic(err)
	} else{
		return strings.Replace(name, r.FindStringSubmatch(name)[0], "_", -1)
	}
}

func findIpAndMac(instance *model.Instance) (string, string, string) {
	for _, nic := range instance.Nics {
		for _, ip := range nic.IPAddresses {
			if ip.Role != "primary" {
				continue
			}
			subnet := fmt.Sprintf("%s/%s", ip.Subnet.NetworkAddress, ip.Subnet.CidrSize)
			return ip.Address, nic.MacAddress, subnet
		}
	}
	return "", "", ""
}