package utils

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/rancher/agent/handlers/marshaller"
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
	//for debug
	d, _ := marshaller.ToString(config["labels"])
	logrus.Info(string(d))
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
	if env, ok := config["enviroment"]; !ok {
		env = make(map[string]string)
		config["enviroment"] = env
	} else {
		env := env.(map[string]interface{})
		for i := 0; i < len(args); i += 2 {
			if _, ok := env[args[i]]; !ok {
				env[args[i]] = args[i+1]
			}
		}
		for key, value := range result {
			if _, ok := env[key]; !ok {
				env[key] = value
			}
		}
	}

}

func getOrCreatePortList(config map[string]interface{}, key string) []model.Port {
	list, ok := config[key]
	if !ok {
		config[key] = list
	}

	return config[key].([]model.Port)
}

func getOrCreateBindingMap(config map[string]interface{}, key string) map[string][]string {
	m, ok := config[key]
	if !ok {
		m = make(map[string]string)
		config[key] = m
	}
	return config[key].(map[string][]string)
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
	return m[key] != nil && len(m[key].([]string)) > 0
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
		os.Mkdir(dstPath, 0777)
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

func GetResponseData(event *events.Event, eventData map[string]interface{}) map[string]interface{} {
	// TODO not implemented
	resourceType := event.ResourceType
	return map[string]interface{}{resourceType: getInstanceHostMapData(event)}
}

func convertPortToString(port int) string {
	if port == 0 {
		return ""
	}
	return strconv.Itoa(port)
}
