package utils

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	engineCli "github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/rancher/agent/progress"
	revents "github.com/rancher/event-subscriber/events"
	v3 "github.com/rancher/go-rancher/v3"
	"golang.org/x/net/context"
)

func GetDeploymentSyncRequest(event *revents.Event) (v3.DeploymentSyncRequest, error) {
	var deploymentSyncRequest v3.DeploymentSyncRequest

	err := Unmarshalling(event.Data["deploymentSyncRequest"], &deploymentSyncRequest)
	if err != nil {
		return v3.DeploymentSyncRequest{}, errors.Wrapf(err, "failed to unmarshall deploymentSyncRequest. Body: %v", event.Data["deploymentSyncRequest"])
	}
	return deploymentSyncRequest, nil
}

func GetContainerSpec(event *revents.Event) (v3.Container, error) {
	var deploymentSyncRequest v3.DeploymentSyncRequest
	err := Unmarshalling(event.Data["deploymentSyncRequest"], &deploymentSyncRequest)
	if err != nil {
		return v3.Container{}, errors.Wrap(err, "failed to unmarshall deploymentSyncRequest")
	}
	if len(deploymentSyncRequest.Containers) == 0 {
		return v3.Container{}, errors.New("the number of instances for deploymentSyncRequest is zero")
	}
	return deploymentSyncRequest.Containers[0], nil
}

func IsNoOp(containerSpec v3.Container) bool {
	if value, ok := GetFieldsIfExist(containerSpec.Data, "processData", "containerNoOpEvent"); ok {
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

func HasLabel(containerSpec v3.Container) bool {
	_, ok := containerSpec.Labels[CattelURLLabel]
	return ok
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

type ContainerNotFoundError struct {
}

func (c ContainerNotFoundError) Error() string {
	return "Container not found"
}

func FindContainer(client *engineCli.Client, containerSpec v3.Container, byAgent bool) (string, error) {
	if containerSpec.ExternalId != "" {
		_, err := client.ContainerInspect(context.Background(), containerSpec.ExternalId)
		if err != nil && !engineCli.IsErrContainerNotFound(err) {
			return "", errors.Wrap(err, "failed to find container from externalId")
		} else if err == nil {
			return containerSpec.ExternalId, nil
		}
	}

	containers, err := client.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		return "", errors.Wrap(err, "failed to list containers")
	}
	for _, cont := range containers {
		if uuid, ok := cont.Labels[UUIDLabel]; ok && uuid == containerSpec.Uuid {
			return cont.ID, nil
		}
	}
	if cont, ok := FindFirst(containers, func(c types.Container) bool {
		if c.Labels[UUIDLabel] == containerSpec.Uuid {
			return true
		}
		return false
	}); ok {
		return cont.ID, nil
	}

	if externalID := containerSpec.ExternalId; externalID != "" {
		if cont, ok := FindFirst(containers, func(c types.Container) bool {
			return IDFilter(externalID, c)
		}); ok {
			return cont.ID, nil
		}
	}

	if byAgent {
		if cont, ok := FindFirst(containers, func(c types.Container) bool {
			return AgentIDFilter(containerSpec.AgentId, c)
		}); ok {
			return cont.ID, nil
		}
	}

	return "", ContainerNotFoundError{}
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
	containerID, ok := container.Labels[AgentIDLabel]
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
		return errors.Wrap(err, "failed to remove container")
	}
	return nil
}

func IsContainerNotFoundError(e error) bool {
	_, ok := e.(ContainerNotFoundError)
	return ok
}

func GetProgress(request *revents.Event, cli *v3.RancherClient) *progress.Progress {
	progress := progress.Progress{
		Request: request,
		Client:  cli,
	}
	return &progress
}

func ToMapString(m map[string]interface{}) map[string]string {
	r := map[string]string{}
	for k, v := range m {
		r[k] = InterfaceToString(v)
	}
	return r
}
