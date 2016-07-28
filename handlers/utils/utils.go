package utils

import (
	"fmt"
	"github.com/docker/go-connections/nat"
	"github.com/rancher/agent/model"
	"github.com/rancher/go-machine-service/events"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strconv"
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

func addLabel(config map[string]interface{}, newLabels map[string]string) {
	labels, ok := config["labels"]
	if !ok {
		labels = make(map[string]string)
		config["labels"] = labels
	}
	for key, value := range newLabels {
		config["labels"].(map[string]string)[key] = value
	}
}

func searchInList(slice []string, target string) bool {
	for _, value := range slice {
		if target == value {
			return true
		}
	}
	return false
}

func defaultValue(name string, df string) string {
	if value, ok := ConfigOverride[name]; ok {
		return value
	}
	if result := os.Getenv(fmt.Sprintf("CATTLE_%s", name)); result != "" {
		return result
	}
	return df
}

func isNonrancherContainer(instance *model.Instance) bool {
	return instance.NativeContainer
}

func addToEnv(config map[string]interface{}, result map[string]string, args ...string) {
	if envs, ok := config["env"]; !ok {
		envs = []string{}
		config["env"] = envs
	}
	envs := config["env"].([]string)
	for key, value := range result {
		envs = append(envs, fmt.Sprintf("%v=%v", key, value))
	}
	config["env"] = envs
}

func getOrCreateBindingMap(config map[string]interface{}, key string) nat.PortMap {
	_, ok := config[key]
	if !ok {
		config[key] = nat.PortMap{}
	}
	return config[key].(nat.PortMap)
}

func hasKey(m interface{}, key string) bool {
	_, ok := m.(map[string]interface{})[key]
	return ok
}

//TODO mock not implemented
func checkOutput(strs []string) {

}

func hasLabel(instance *model.Instance) bool {
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

func isStrSet(m map[string]interface{}, key string) bool {
	ok := false
	switch m[key].(type) {
	case string:
		ok = len(m[key].(string)) > 0
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

func tempFileInWorkDir(destination string) string {
	dstPath := path.Join(destination, TempName)
	if _, err := os.Stat(dstPath); os.IsNotExist(err) {
		os.MkdirAll(dstPath, 0777)
	}
	return tempFile(dstPath)
}

func tempFile(destination string) string {
	tempDst, err := ioutil.TempFile(destination, TempPrefix)
	if err == nil {
		return tempDst.Name()
	}
	return ""
}

func GetResponseData(event *events.Event) map[string]interface{} {
	resourceType := event.ResourceType
	switch resourceType {
	case "instanceHostMap":
		return map[string]interface{}{resourceType: getInstanceHostMapData(event)}
	case "volumeStoragePoolMap":
		return map[string]interface{}{
			resourceType: map[string]interface{}{
				"volume": map[string]interface{}{
					"format": "docker",
				},
			},
		}
	case "instancePull":
		return map[string]interface{}{
			"fields": map[string]interface{}{
				"dockerImage": getInstancePullData(event),
			},
		}
	default:
		return map[string]interface{}{resourceType: map[string]interface{}{}}
	}

}

func convertPortToString(port int) string {
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
