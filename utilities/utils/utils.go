package utils

import (
	"encoding/json"
	"fmt"
	engineCli "github.com/docker/docker/client"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/rancher/agent/core/progress"
	"github.com/rancher/agent/model"
	"github.com/rancher/agent/utilities/constants"
	"github.com/rancher/agent/utilities/docker"
	revents "github.com/rancher/event-subscriber/events"
	"github.com/rancher/go-rancher/v2"
	"golang.org/x/net/context"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func GetInstanceAndHost(event *revents.Event) (model.Instance, model.Host, error) {

	data := event.Data
	var ihm model.InstanceHostMap
	if err := mapstructure.Decode(data["instanceHostMap"], &ihm); err != nil {
		return model.Instance{}, model.Host{}, errors.Wrap(err, constants.GetInstanceAndHostError+"failed to marshall instancehostmap")
	}

	var instance model.Instance
	if err := mapstructure.Decode(ihm.Instance, &instance); err != nil {
		return model.Instance{}, model.Host{}, errors.Wrap(err, constants.GetInstanceAndHostError+"failed to marshall instance data")
	}
	var host model.Host
	if err := mapstructure.Decode(ihm.Host, &host); err != nil {
		return model.Instance{}, model.Host{}, errors.Wrap(err, constants.GetInstanceAndHostError+"failed to marshall host data")
	}

	return instance, host, nil
}

func IsNoOp(data model.ProcessData) bool {
	return data.ContainerNoOpEvent
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

func IsNonrancherContainer(instance model.Instance) bool {
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

func HasLabel(instance model.Instance) bool {
	_, ok := instance.Labels[constants.CattelURLLabel]
	return ok
}

func ReadBuffer(reader io.ReadCloser) string {
	buffer := make([]byte, 1024)
	s := ""
	for {
		n, err := reader.Read(buffer)
		s = s + string(buffer[:n])
		if err != nil {
			break
		}
	}
	return s
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

func HasService(instance model.Instance, kind string) bool {
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

	ports := link.Data.Fields.Ports
	for _, port := range ports {
		protocol := port.Protocol
		ip := strings.ToLower(name)
		if inIP != "" {
			ip = inIP
		}
		dst := port.PrivatePort
		src := port.PrivatePort

		fullPort := fmt.Sprintf("%v://%v:%v", protocol, ip, dst)
		data := make(map[string]string)
		data["NAME"] = fmt.Sprintf("/cattle/%v", name)
		data["PORT"] = fullPort
		data[fmt.Sprintf("PORT_%v_%v", src, protocol)] = fullPort
		data[fmt.Sprintf("PORT_%v_%v_ADDR", src, protocol)] = ip
		data[fmt.Sprintf("PORT_%v_%v_PORT", src, protocol)] = getStringOrFloat(dst)
		data[fmt.Sprintf("PORT_%v_%v_PROTO", src, protocol)] = protocol
		for key, value := range data {
			result[strings.ToUpper(fmt.Sprintf("%v_%v", toEnvName(name), key))] = value
		}
	}

}

func CopyLinkEnv(name string, link model.Link, result map[string]string) {
	targetInstance := link.TargetInstance
	envs := targetInstance.Data.DockerInspect.Config.Env
	ignores := make(map[string]bool)
	for _, env := range envs {
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
	for _, env := range envs {
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

func toEnvName(name string) string {
	r, _ := regexp.Compile("[^a-zA-Z0-9_]")
	if r.FindStringSubmatch(name) != nil {
		name = strings.Replace(name, r.FindStringSubmatch(name)[0], "_", -1)
	}
	return strings.ToUpper(name)
}

func FindIPAndMac(instance model.Instance) (string, string, string) {
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

func ParseRepoTag(name string) string {
	if strings.HasPrefix(name, "docker:") {
		name = name[7:]
	}
	return name
}

func GetContainer(client *engineCli.Client, instance model.Instance, byAgent bool) (types.Container, error) {
	clientWithTimeout, err := docker.NewEnvClientWithTimeout(time.Duration(2) * time.Second)
	if err != nil {
		return types.Container{}, errors.Wrap(err, constants.GetContainerError+"failed to get docker client")
	}
	clientWithTimeout.UpdateClientVersion(constants.DefaultVersion)
	containers, err := clientWithTimeout.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		return types.Container{}, errors.Wrap(err, constants.GetContainerError+"failed to list containers")
	}
	for _, container := range containers {
		if uuid, ok := container.Labels[constants.UUIDLabel]; ok && uuid == instance.UUID {
			return container, nil
		}
	}
	if container, ok := FindFirst(containers, func(c types.Container) bool {
		if GetUUID(c) == instance.UUID {
			return true
		}
		return false
	}); ok {
		return container, nil
	}

	if externalID := instance.ExternalID; externalID != "" {
		if container, ok := FindFirst(containers, func(c types.Container) bool {
			return IDFilter(externalID, c)
		}); ok {
			return container, nil
		}
	}

	if byAgent {
		agentID := instance.AgentID
		if container, ok := FindFirst(containers, func(c types.Container) bool {
			return AgentIDFilter(strconv.Itoa(agentID), c)
		}); ok {
			return container, nil
		}
	}

	return types.Container{}, model.ContainerNotFoundError{}
}

func GetUUID(container types.Container) string {
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

func FindFirst(containers []types.Container, f func(types.Container) bool) (types.Container, bool) {
	for _, c := range containers {
		if f(c) {
			return c, true
		}
	}
	return types.Container{}, false
}

func IDFilter(id string, container types.Container) bool {
	return container.ID == id
}

func AgentIDFilter(id string, container types.Container) bool {
	containerID, ok := container.Labels[constants.AgentIDLabel]
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

func NameFilter(name string, container types.Container) bool {
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
	if err := client.ContainerRemove(context.Background(), containerID, types.ContainerRemoveOptions{}); !engineCli.IsErrContainerNotFound(err) {
		return errors.Wrap(err, constants.RemoveContainerError+"failed to remove container")
	}
	return nil
}

func AddContainer(state string, container types.Container, containers []model.PingResource, dockerClient *engineCli.Client) []model.PingResource {
	containerData := model.PingResource{
		Type:     "instance",
		UUID:     GetUUID(container),
		State:    state,
		DockerID: container.ID,
		Image:    container.Image,
		Labels:   container.Labels,
		Created:  container.Created,
	}
	return append(containers, containerData)
}

func Get(url string) (map[string]interface{}, error) {
	resp, err := http.Get(url)
	if err == nil {
		defer resp.Body.Close()
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, errors.Wrap(err, constants.GetCadvisorError)
		}
		var result map[string]interface{}
		err1 := json.Unmarshal(data, &result)
		if err1 != nil {
			return nil, errors.Wrap(err, constants.GetCadvisorError)
		}
		return result, nil
	}
	return nil, errors.Wrap(err, constants.GetCadvisorError)
}

func IsContainerNotFoundError(e error) bool {
	_, ok := e.(model.ContainerNotFoundError)
	return ok
}

func IsImageNoOp(imageData model.ImageData) bool {
	return imageData.ProcessData.ContainerNoOpEvent
}

func IsPathExist(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}

func GetProgress(request *revents.Event, cli *client.RancherClient) *progress.Progress {
	progress := progress.Progress{
		Request: request,
		Client:  cli,
	}
	return &progress
}

//weird method to convert an interface to string
func getStringOrFloat(v interface{}) string {
	if f, ok := v.(float64); ok {
		return strconv.FormatFloat(f, 'f', -1, 64)
	}
	return v.(string)
}

func GetExitCode(err error) int {
	if exitError, ok := err.(*exec.ExitError); ok {
		status := exitError.Sys().(syscall.WaitStatus)
		return status.ExitStatus()
	}
	return -1
}
