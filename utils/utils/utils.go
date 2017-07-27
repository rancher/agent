package utils

import (
	"fmt"
	engineCli "github.com/docker/docker/client"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/rancher/agent/core/progress"
	"github.com/rancher/agent/model"
	"github.com/rancher/agent/utils/constants"
	revents "github.com/rancher/event-subscriber/events"
	"github.com/rancher/go-rancher/v2"
	"golang.org/x/net/context"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func GetInstance(event *revents.Event) (model.Instance, error) {
	data := event.Data
	var instance model.Instance
	if err := mapstructure.Decode(data["instance"], &instance); err != nil {
		return model.Instance{}, errors.Wrap(err, constants.GetInstanceAndHostError+"failed to marshall instancehostmap")
	}
	return instance, nil
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

func AddToEnv(config *container.Config, result map[string]string, args ...string) {
	envs := config.Env
	existKeys := map[string]struct{}{}
	for _, key := range envs {
		parts := strings.Split(key, "=")
		if len(parts) > 0 {
			existKeys[parts[0]] = struct{}{}
		}
	}
	for key, value := range result {
		if _, ok := existKeys[key]; !ok {
			envs = append(envs, fmt.Sprintf("%v=%v", key, value))
		}
	}
	config.Env = envs
}

func HasKey(m interface{}, key string) bool {
	_, ok := m.(map[string]interface{})[key]
	return ok
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

func ParseRepoTag(name string) string {
	if strings.HasPrefix(name, "docker:") {
		name = name[7:]
	}
	return name
}

func GetContainer(client *engineCli.Client, instance model.Instance, byAgent bool) (types.Container, error) {
	containers, err := client.ContainerList(context.Background(), types.ContainerListOptions{All: true})
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

func IsContainerNotFoundError(e error) bool {
	_, ok := e.(model.ContainerNotFoundError)
	return ok
}

func GetProgress(request *revents.Event, cli *client.RancherClient) *progress.Progress {
	progress := progress.Progress{
		Request: request,
		Client:  cli,
	}
	return &progress
}

func GetExitCode(err error) int {
	if exitError, ok := err.(*exec.ExitError); ok {
		status := exitError.Sys().(syscall.WaitStatus)
		return status.ExitStatus()
	}
	return -1
}

func IsNoopEvent(event *revents.Event) bool {
	if noOp, ok := GetFieldsIfExist(event.Data, "processData", "containerNoOpEvent"); ok {
		return InterfaceToBool(noOp)
	}
	return false
}
