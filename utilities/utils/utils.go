package utils

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/Sirupsen/logrus"
	engineCli "github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/container"
	"github.com/docker/engine-api/types/filters"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/rancher/agent/model"
	"github.com/rancher/agent/utilities/constants"
	"github.com/rancher/agent/utilities/docker"
	revents "github.com/rancher/event-subscriber/events"
	"golang.org/x/net/context"
	"io"
	"io/ioutil"
	"net/http"
	urls "net/url"
	"os"
	"os/exec"
	"path"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
<<<<<<< 9b5369dc8d23c302070697f35f70ded5f640c0c4
	"os/exec"
=======
>>>>>>> fmt correct
)

func unwrap(obj interface{}) interface{} {
	switch obj.(type) {
	case []map[string]interface{}:
		ret := []map[string]interface{}{}
		obj := []map[string]interface{}{}
		for _, o := range obj {
			ret = append(ret, unwrap(o).(map[string]interface{}))
		}
		return ret
	case map[string]interface{}:
		ret := map[string]interface{}{}
		obj := map[string]interface{}{}
		for key, value := range obj {
			ret[key] = unwrap(value)
		}
		return ret
	default:
		return obj
	}
}

func GetInstanceAndHost(event *revents.Event) (*model.Instance, *model.Host) {

	data := event.Data
	var ihm model.InstanceHostMap
	mapstructure.Decode(data["instanceHostMap"], &ihm)

	var instance model.Instance
	mapstructure.Decode(ihm.Instance, &instance)
	var host model.Host
	mapstructure.Decode(ihm.Host, &host)

	clusterConnection, ok := GetFieldsIfExist(data, "field", "clusterConnection")
	if ok {
		host.Data["clusterConnection"] = InterfaceToString(clusterConnection)
		if strings.HasPrefix(InterfaceToString(clusterConnection), "http") {
			caCrt, ok1 := GetFieldsIfExist(event.Data, "field", "caCrt")
			clientCrt, ok2 := GetFieldsIfExist(event.Data, "field", "clientCrt")
			clientKey, ok3 := GetFieldsIfExist(event.Data, "field", "clientKey")
			// what if we miss certs/key? do we have to panic or ignore it?
			if ok1 && ok2 && ok3 {
				host.Data["caCrt"] = InterfaceToString(caCrt)
				host.Data["clientCrt"] = InterfaceToString(clientCrt)
				host.Data["clientKey"] = InterfaceToString(clientKey)
			} else {
				logrus.Infof("Missing certs/key [%v]for clusterConnection for connection ",
					clusterConnection)
			}
		}
	}
	return &instance, &host
}

func IsNoOp(data map[string]interface{}) bool {
	b, ok := GetFieldsIfExist(data, "processData", "containerNoOpEvent")
	if ok {
		return InterfaceToBool(b)
	}
	return false
}

func IsTrue(instance *model.Instance, field string) bool {
	value, ok := GetFieldsIfExist(instance.Data, "fields", field)
	if ok {
		return InterfaceToBool(value)
	}
	return false
}

func AddLabel(config *container.Config, key string, value string) {
	config.Labels[key] = value
}

func SearchInList(slice []string, target string) bool {
	for _, value := range slice {
		if target == value {
			return true
		}
	}
	return false
}

func IsNonrancherContainer(instance *model.Instance) bool {
	return instance.NativeContainer
}

func AddToEnv(config *container.Config, result map[string]string, args ...string) {
	envs := config.Env
	for key, value := range result {
		envs = append(envs, fmt.Sprintf("%v=%v", key, value))
	}
	config.Env = envs
}

func HasKey(m interface{}, key string) bool {
	_, ok := m.(map[string]interface{})[key]
	return ok
}

func CheckOutput(strs []string) {

}

func HasLabel(instance *model.Instance) bool {
	_, ok := instance.Labels["io.rancher.container.cattle_url"]
	return ok
}

func ReadBuffer(reader io.ReadCloser) string {
	buffer := make([]byte, 1024)
	s := ""
	defer reader.Close()
	for {
		n, err := reader.Read(buffer)
		s = s + string(buffer[:n])
		if err != nil {
			break
		}
	}
	return s
}

func IsStrSet(m map[string]interface{}, key string) bool {
	ok := false
	switch m[key].(type) {
	case string:
		ok = len(InterfaceToString(m[key])) > 0
	case []string:
		ok = len(m[key].([]string)) > 0
	}
	return m[key] != nil && ok
}

func GetFieldsIfExist(m map[string]interface{}, fields ...string) (interface{}, bool) {
	var tempMap map[string]interface{}
	tempMap = m
	for i, field := range fields {
		switch tempMap[field].(type) {
		case map[string]interface{}:
			tempMap = tempMap[field].(map[string]interface{})
			break
		case nil:
			return nil, false
		default:
			// if it is the last field and it is not empty
			// it exists othewise return false
			if i == len(fields)-1 {
				return tempMap[field], true
			}
			return nil, false
		}
	}
	return tempMap, true
}

func TempFileInWorkDir(destination string) string {
	dstPath := path.Join(destination, constants.TempName)
	if _, err := os.Stat(dstPath); os.IsNotExist(err) {
		os.MkdirAll(dstPath, 0777)
	}
	return TempFile(dstPath)
}

func TempFile(destination string) string {
	tempDst, err := ioutil.TempFile(destination, constants.TempPrefix)
	if err == nil {
		return tempDst.Name()
	}
	return ""
}

func ConvertPortToString(port int) string {
	if port == 0 {
		return ""
	}
	return strconv.Itoa(port)
}

func InterfaceToString(v interface{}) string {
	value, ok := v.(string)
	if ok {
		return value
	}
	return ""
}

func InterfaceToInt(v interface{}) int {
	value, ok := v.(int)
	if ok {
		return value
	}
	return 0
}

func InterfaceToBool(v interface{}) bool {
	value, ok := v.(bool)
	if ok {
		return value
	}
	return false
}

func InterfaceToFloat(v interface{}) float64 {
	value, ok := v.(float64)
	if ok {
		return value
	}
	return 0.0
}

func InterfaceToArray(v interface{}) []interface{} {
	value, ok := v.([]interface{})
	if ok {
		return value
	}
	return []interface{}{}
}

func InterfaceToMap(v interface{}) map[string]interface{} {
	value, ok := v.(map[string]interface{})
	if ok {
		return value
	}
	return map[string]interface{}{}
}

func SafeSplit(s string) []string {
	split := strings.Split(s, " ")

	var result []string
	var inquote string
	var block string
	for _, i := range split {
		if inquote == "" {
			if strings.HasPrefix(i, "'") || strings.HasPrefix(i, "\"") {
				inquote = string(i[0])
				block = strings.TrimPrefix(i, inquote) + " "
			} else {
				result = append(result, i)
			}
		} else {
			if !strings.HasSuffix(i, inquote) {
				block += i + " "
			} else {
				block += strings.TrimSuffix(i, inquote)
				inquote = ""
				result = append(result, block)
				block = ""
			}
		}
	}

	return result
}

func HasService(instance *model.Instance, kind string) bool {
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

func AddLinkEnv(name string, link model.Link, result map[string]string, inIP string) {
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
			data[fmt.Sprintf("PORT_%v_%v_PORT", src, protocol)] = InterfaceToString(dst)
			data[fmt.Sprintf("PORT_%v_%v_PROTO", src, protocol)] = InterfaceToString(protocol)
			for key, value := range data {
				result[strings.ToUpper(fmt.Sprintf("%v_%v", toEnvName(name), key))] = value
			}
		}
	}
}

func CopyLinkEnv(name string, link model.Link, result map[string]string) {
	targetInstance := link.TargetInstance
	if envs, ok := GetFieldsIfExist(targetInstance.Data, "dockerInspect", "Config", "Env"); ok {
		ignores := make(map[string]bool)
		for _, env := range envs.([]interface{}) {
			env := InterfaceToString(env)
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
			env := InterfaceToString(env)
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

func FindIPAndMac(instance *model.Instance) (string, string, string) {
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

func ParseRepoTag(name string) map[string]string {
	if strings.HasPrefix(name, "docker:") {
		name = name[7:]
	}
	n := strings.Index(name, ":")
	if n < 0 {
		return map[string]string{
			"repo": name,
			"tag":  "latest",
			"uuid": name + ":latest",
		}
	}
	tag := name[n+1:]
	if strings.Index(tag, "/") < 0 {
		return map[string]string{
			"repo": name[:n],
			"tag":  tag,
			"uuid": name,
		}
	}
	return map[string]string{
		"repo": name,
		"tag":  "latest",
		"uuid": name + ":latest",
	}
}

func GetContainer(client *engineCli.Client, instance *model.Instance, byAgent bool) *types.Container {
	if instance == nil {
		return nil
	}

	// First look for UUID label directly
	args := filters.NewArgs()
	args.Add("label", fmt.Sprintf("%s=%s", constants.UUIDLabel, instance.UUID))
	options := types.ContainerListOptions{All: true, Filter: args}
	labeledContainers, err := client.ContainerList(context.Background(), options)
	if err == nil && len(labeledContainers) > 0 {
		return &labeledContainers[0]
	}

	// Next look by UUID using fallback method
	options = types.ContainerListOptions{All: true}
	containerList, err := client.ContainerList(context.Background(), options)
	if err != nil {
		return nil
	}
	container := FindFirst(containerList, func(c *types.Container) bool {
		if GetUUID(c) == instance.UUID {
			return true
		}
		return false
	})

	if container != nil {
		return container
	}
	if externalID := instance.ExternalID; externalID != "" {
		container = FindFirst(containerList, func(c *types.Container) bool {
			return IDFilter(externalID, c)
		})
	}

	if container != nil {
		return container
	}

	if byAgent {
		agentID := instance.AgentID
		container = FindFirst(containerList, func(c *types.Container) bool {
			return AgentIDFilter(strconv.Itoa(agentID), c)
		})
	}

	return container
}

func GetUUID(container *types.Container) string {
	if uuid, ok := container.Labels[constants.UUIDLabel]; ok {
		return uuid
	}

	names := container.Names
	if len(names) == 0 {
		return fmt.Sprintf("no-uuid-%s", container.ID)
	}

	if strings.HasPrefix(names[0], "/") {
		return names[0][1:]
	}
	return names[0]
}

func FindFirst(containers []types.Container, f func(*types.Container) bool) *types.Container {
	for _, c := range containers {
		if f(&c) {
			return &c
		}
	}
	return nil
}

func IDFilter(id string, container *types.Container) bool {
	return container.ID == id
}

func AgentIDFilter(id string, container *types.Container) bool {
	containerID, ok := container.Labels["io.rancher.container.agent_id"]
	if ok {
		return containerID == id
	}
	return false
}

func SemverTrunk(version string, vals int) string {
	/*
			vrm_vals: is a number representing the number of
		        digits to return. ex: 1.8.3
		          vmr_val = 1; return val 1
		          vmr_val = 2; return val 1.8
		          vmr_val = 3; return val 1.8.3
	*/
	if version != "" {
		m := map[int]string{
			1: regexp.MustCompile("(\\d+)").FindString(version),
			2: regexp.MustCompile("(\\d+\\.)?(\\d+)").FindString(version),
			3: regexp.MustCompile("(\\d+\\.)?(\\d+\\.)?(\\d+)").FindString(version),
		}
		return m[vals]
	}
	return version
}

func GetKernelVersion() string {
	if runtime.GOOS == "linux" {
		file, err := os.Open("/proc/version")
		defer file.Close()
		data := []string{}
		if err != nil {
			logrus.Error(err)
		} else {
			scanner := bufio.NewScanner(file)
			scanner.Split(bufio.ScanLines)
			for scanner.Scan() {
				data = append(data, scanner.Text())
			}
		}
		version := regexp.MustCompile("\\d+.\\d+.\\d+").FindString(data[0])
		return version
	}
	return ""
}

func GetLoadAverage() []string {
	if runtime.GOOS == "linux" {
		file, err := os.Open("/proc/loadavg")
		defer file.Close()
		data := []string{}
		if err != nil {
			logrus.Error(err)
		} else {
			scanner := bufio.NewScanner(file)
			scanner.Split(bufio.ScanLines)
			for scanner.Scan() {
				data = append(data, scanner.Text())
			}
		}
		loads := strings.Split(data[0], " ")
		return loads[:3]
	}
	return []string{}
}

func GetInfo() (types.Info, error) {
	return docker.Info, docker.InfoErr
}

func NameFilter(name string, container *types.Container) bool {
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

func RemoveContainer(client *engineCli.Client, containerID string) error {
	client.ContainerKill(context.Background(), containerID, "KILL")
	for i := 0; i < 10; i++ {
		if inspect, err := client.ContainerInspect(context.Background(), containerID); err == nil && inspect.State.Pid == 0 {
			break
		}
		time.Sleep(time.Duration(500) * time.Millisecond)
	}
	if err := client.ContainerRemove(context.Background(), containerID, types.ContainerRemoveOptions{}); !engineCli.IsErrNotFound(err) {
		return errors.Wrap(err, "failed to remove container")
	}
	return nil
}

func AddContainer(state string, container *types.Container, containers []map[string]interface{}) []map[string]interface{} {
	labels := container.Labels
	containerData := map[string]interface{}{
		"type":            "instance",
		"uuid":            GetUUID(container),
		"state":           state,
		"systemContainer": getSysContainer(container),
		"dockerId":        container.ID,
		"image":           container.Image,
		"labels":          labels,
		"created":         container.Created,
	}
	return append(containers, containerData)
}

func getSysContainer(container *types.Container) string {
	image := container.Image
	systemImages := getAgentImage()
	if HasKey(systemImages, image) {
		return InterfaceToString(systemImages[image])
	}
	label, ok := container.Labels["io.rancher.container.system"]
	if ok {
		return label
	}
	return ""
}

func getAgentImage() map[string]interface{} {
	client := docker.DefaultClient
	args := filters.NewArgs()
	args.Add("label", constants.SystemLables)
	images, _ := client.ImageList(context.Background(), types.ImageListOptions{Filters: args})
	systemImage := map[string]interface{}{}
	for _, image := range images {
		labelValue := image.Labels[constants.SystemLables]
		for _, l := range image.RepoTags {
			if strings.HasSuffix(l, ":latest") {
				alias := l[:len(l)-7]
				systemImage[alias] = labelValue
			}
		}
	}
	return systemImage
}

func Get(url string) (map[string]interface{}, error) {
	resp, err := http.Get(url)
	if err == nil {
		defer resp.Body.Close()
		data, _ := ioutil.ReadAll(resp.Body)
		var result map[string]interface{}
		err1 := json.Unmarshal(data, &result)
		if err1 != nil {
			logrus.Error(err1)
		}
		return result, nil
	}
	return nil, err
}

func GetInfoDriver() string {
	if info, err := GetInfo(); err != nil {
		return info.Driver
	}
	return ""
}

func DockerVersionRequest() (types.Version, error) {
	client := docker.DefaultClient
	return client.ServerVersion(context.Background())
}

func GetURLPort(url string) string {
	parse, err := urls.Parse(url)
	if err != nil {
		return ""
	}
	host := parse.Host
	parts := strings.Split(host, ":")
	if len(parts) == 2 {
		return parts[1]
	}
	port := ""
	if parse.Scheme == "http" {
		port = "80"
	} else if parse.Scheme == "https" {
		port = "443"
	}
	return port
}

func GetWindowsKernelVersion() (string, error) {
	command := exec.Command("PowerShell", "wmic", "os", "get", "Version")
	output, err := command.Output()
	if err == nil {
		ret := strings.Split(string(output), "\n")[1]
		return ret, nil
	} else {
		return "", err
	}
}
