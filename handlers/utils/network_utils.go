package utils

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/docker/engine-api/client"
	nat1 "github.com/docker/go-connections/nat"
	"github.com/mitchellh/mapstructure"
	"github.com/rancher/agent/model"
	"regexp"
	"strconv"
	"strings"
)

func setupMacAndIP(instance *model.Instance, createConfig map[string]interface{}, setMac bool, setHostname bool) {
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
	logrus.Infof("macAddress :%v", macAddress)
	if setMac {
		createConfig["macAddress"] = macAddress
	}

	if !setHostname {
		delete(createConfig, "hostname")
	}

	if instance.Nics != nil && len(instance.Nics) > 0 && instance.Nics[0].IPAddresses != nil {
		// Assume one nic
		nic := instance.Nics[0]
		logrus.Info("nic info %v", nic)
		ipAddress := ""
		for _, ip := range nic.IPAddresses {
			logrus.Info("ip info %v", ip)
			if ip.Role == "primary" {
				ipAddress = fmt.Sprintf("%s/%s", ip.Address, strconv.Itoa(ip.Subnet.CidrSize))
				break
			}
		}
		logrus.Info("ip info %s", ipAddress)
		if ipAddress != "" {
			addLabel(createConfig, map[string]string{"io.rancher.container.ip": ipAddress})
		}
	}
}

func setupNetworkMode(instance *model.Instance, client *client.Client,
	createConfig map[string]interface{}, startConfig map[string]interface{}) (bool, bool) {
	/*
			Based on the network configuration we choose the network mode to set in
		    Docker.  We only really look for none, host, or container.  For all
		    all other configurations we assume bridge mode
	*/
	portsSupported := true
	hostnameSupported := true
	if len(instance.Nics) > 0 {
		kind := instance.Nics[0].Network.Kind
		if kind == "dockermodel.model.Host" {
			portsSupported = false
			hostnameSupported = false
			startConfig["network_mode"] = "host"
			delete(startConfig, "link")
		} else if kind == "dockerNone" {
			portsSupported = false
			createConfig["network_mode"] = "none"
			delete(startConfig, "link")
		} else if kind == "dockerContainer" {
			portsSupported = false
			hostnameSupported = false
			id := instance.NetworkContainer["uuid"]
			var in model.Instance
			mapstructure.Decode(instance.NetworkContainer, &in)
			other := GetContainer(client, &in, false)
			if other != nil {
				id = other.ID
			}
			startConfig["network_mode"] = fmt.Sprintf("container:%s", id)
			delete(startConfig, "link")
		}
	}
	return portsSupported, hostnameSupported

}

func setupPortsNetwork(instance *model.Instance, createConfig map[string]interface{},
	startConfig map[string]interface{}, portsSupported bool) {
	/*
			Docker 1.9+ does not allow you to pass port info for networks that don't
		    support ports (net, none, container:x)
	*/
	if !portsSupported {
		startConfig["publish_all_ports"] = false
		delete(createConfig, "ports")
		delete(startConfig, "port_bindings")
	}
}

func setupIpsec(instance *model.Instance, host *model.Host, createConfig map[string]interface{},
	startConfig map[string]interface{}) {
	/*
			If the supplied instance is a network agent, configures the ports needed
		    to achieve multi-host networking.
	*/
	networkAgent := false
	if instance.SystemContainer == "" || instance.SystemContainer == "NetworkAgent" {
		networkAgent = true
	}
	if !networkAgent || !hasService(instance, "ipsecTunnelService") {
		return
	}
	hostID := strconv.Itoa(host.ID)
	if info, ok := instance.Data["ipsec"].(map[string]interface{})[hostID].(map[string]interface{}); ok {
		nat := info["nat"].(float64)
		isakmp := info["isakmp"].(float64)

		binding := getOrCreateBindingMap(startConfig, "portbindings")

		port1 := nat1.Port(fmt.Sprintf("%v/%v", 500, "udp"))
		port2 := nat1.Port(fmt.Sprintf("%v/%v", 4500, "udp"))
		bind1 := nat1.PortBinding{HostIP: "0.0.0.0", HostPort: strconv.Itoa(int(isakmp))}
		bind2 := nat1.PortBinding{HostIP: "0.0.0.0", HostPort: strconv.Itoa(int(nat))}
		exposedPorts := map[nat1.Port]struct{}{
			port1: struct{}{},
			port2: struct{}{},
		}
		if _, ok := binding[port1]; ok {
			binding[port1] = append(binding[port1], bind1)
		} else {
			binding[port1] = []nat1.PortBinding{bind1}
		}
		if _, ok := binding[port2]; ok {
			binding[port2] = append(binding[port2], bind1)
		} else {
			binding[port2] = []nat1.PortBinding{bind2}
		}
		if _, ok := createConfig["exposedPorts"]; ok {
			existingMap := createConfig["exposedPorts"].(map[nat1.Port]struct{})
			for port := range exposedPorts {
				existingMap[port] = struct{}{}
			}
			createConfig["exposedPorts"] = existingMap
		} else {
			createConfig["exposedPorts"] = exposedPorts
		}

	}
}

func setupDNS(instance *model.Instance) {
	if !hasService(instance, "dnsService") || instance.Kind == "virtualMachine" {
		return
	}
	ipAddress, macAddress, subnet := findIPAndMac(instance)

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

	//TODO implement check_output function

	checkOutput([]string{"iptables", "-w", "-t", "nat", "-A", "CATTLE_PREROUTING",
		"!", "-s", subnet, "-d", "169.254.169.250", "-m", "mac",
		"--mac-source", macAddress, "-j", "MARK", "--set-mark",
		mark})
	checkOutput([]string{"iptables", "-w", "-t", "nat", "-A", "CATTLE_POSTROUTING",
		"!", "-s", subnet, "-d", "169.254.169.250", "-m", "mark", "--mark", mark,
		"-j", "SNAT", "--to", ipAddress})

}

func setupLinksNetwork(instance *model.Instance, createConfig map[string]interface{},
	startConfig map[string]interface{}) {
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
	if !hasService(instance, "linkService") || isNonrancherContainer(instance) {
		return
	}

	if hasKey(startConfig, "links") {
		delete(startConfig, "links")
	}
	result := map[string]string{}
	if instance.InstanceLinks != nil {
		for _, link := range instance.InstanceLinks {
			linkName := link.LinkName
			addLinkEnv(linkName, link, result, "")
			copyLinkEnv(linkName, link, result)
			if names, ok := GetFieldsIfExist(link.Data, "fields", "instanceNames"); ok {
				for _, name := range names.([]interface{}) {
					name := name.(string)
					addLinkEnv(name, link, result, linkName)
					copyLinkEnv(name, link, result)
					// This does assume the format {env}_{name}
					parts := strings.SplitN(name, "_", 2)
					if len(parts) == 1 {
						continue
					}
					addLinkEnv(parts[1], link, result, linkName)
					copyLinkEnv(parts[1], link, result)
				}

			}
		}
		if len(result) > 0 {
			addToEnv(createConfig, result)
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

func addLinkEnv(name string, link model.Link, result map[string]string, inIP string) {
	result[strings.ToUpper(fmt.Sprintf("%s_NAME", toEnvName(name)))] = fmt.Sprintf("/cattle/%s", name)

	if ports, ok := GetFieldsIfExist(link.Data, "fields", "ports"); ok {
		for _, port := range ports.([]interface{}) {
			port := port.(map[string]interface{})
			protocol := port["protocol"]
			ip := strings.ToLower(name)
			if inIP != "" {
				ip = inIP
			}
			// different with python agent
			dst := port["privatePort"]
			src := port["privatePort"]

			fullPort := fmt.Sprintf("%v://%v:%v", protocol, ip, dst)
			data := make(map[string]string)
			data["NAME"] = fmt.Sprintf("/cattle/%v", name)
			data["PORT"] = fullPort
			data[fmt.Sprintf("PORT_%v_%v", src, protocol)] = fullPort
			data[fmt.Sprintf("PORT_%v_%v_ADDR", src, protocol)] = ip
			data[fmt.Sprintf("PORT_%v_%v_PORT", src, protocol)] = dst.(string)
			data[fmt.Sprintf("PORT_%v_%v_PROTO", src, protocol)] = protocol.(string)
			logrus.Infof("data map %v", data)
			for key, value := range data {
				result[strings.ToUpper(fmt.Sprintf("%v_%v", toEnvName(name), key))] = value
			}
		}
	}
}

func copyLinkEnv(name string, link model.Link, result map[string]string) {
	targetInstance := link.TargetInstance
	if envs, ok := GetFieldsIfExist(targetInstance.Data, "dockerInspect", "Config", "Env"); ok {
		ignores := make(map[string]bool)
		for _, env := range envs.([]interface{}) {
			env := env.(string)
			logrus.Info(env)
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 1 {
				continue
			}
			if strings.HasPrefix(parts[1], "/cattle/") {
				envName := toEnvName(parts[1][len("/cattle/"):])
				ignores[envName+"_NAME"] = true
				ignores[envName+"_PORT"] = true
				ignores[envName+"_ENV"] = true
			}
		}
		for _, env := range envs.([]interface{}) {
			env := env.(string)
			shouldIgnore := false
			for ignore := range ignores {
				if strings.HasPrefix(env, ignore) {
					shouldIgnore = true
					break
				}
			}
			if shouldIgnore {
				continue
			}
			parts := strings.SplitN(env, "=", 2)
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
	r, _ := regexp.Compile("[^a-zA-Z0-9_]")
	if r.FindStringSubmatch(name) != nil {
		name = strings.Replace(name, r.FindStringSubmatch(name)[0], "_", -1)
	}
	return strings.ToUpper(name)
}

func findIPAndMac(instance *model.Instance) (string, string, string) {
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
