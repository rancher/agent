package utils

import (
	"os"
	"fmt"
	"io"
	"path"
	"net/http"
	"github.com/Sirupsen/logrus"
	"github.com/rancher/agent/model"
	"io/ioutil"
	"github.com/rancher/go-machine-service/events"
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

func add_label(config map[string]interface{}, new_labels map[string]string){
	labels, ok := config["labels"]
	if !ok {
		labels = make(map[string]string)
		config["labels"] = labels
	}
	update(config, new_labels)
}

func update(config map[string]interface{}, new_labels map[string]string){
	for key, value := range new_labels {
		config["labels"].(map[string]string)[key] = value
	}
}

func search_in_list(slice []string, target string) bool {
	for _, value := range slice {
		if target == value {
			return true
		}
	}
	return false
}

func default_value(name string, df string) string {
	if value, ok := CONFIG_OVERRIDE[name]; ok {
		return value
	}
	if result := os.Getenv(fmt.Sprintf("CATTLE_%s", name)); result != "" {
		return result
	}
	return df
}

func is_nonrancher_container(instance *model.Instance) bool {
	return instance.NativeContainer
}

func add_to_env(config map[string]interface{}, result map[string]string, args ...string){
	if env, ok := config["enviroment"]; !ok {
		env = make(map[string]string)
		config["enviroment"] = env
	} else {
		env := env.(map[string]interface{})
		for i := 0 ; i < len(args) ; i += 2 {
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

func get_or_create_port_list(config map[string]interface{}, key string) []model.Port {
	list, ok := config[key]
	if !ok {
		config[key] = list
	}

	return config[key].([]model.Port)
}

func get_or_create_binding_map(config map[string]interface{}, key string) map[string][]string {
	m, ok := config[key]
	if !ok {
		m = make(map[string]string)
		config[key] = m
	}
	return config[key].(map[string][]string)
}

func has_key(m interface{}, key string) bool {
	_, ok := m.(map[string]interface{})[key]
	return ok
}

//TODO mock not implemented
func check_output(strs []string){

}

func has_label(instance *model.Instance) bool{
	_, ok := instance.Labels["io.rancher.container.cattle_url"]
	return ok
}

func readBuffer(reader io.ReadCloser) string {
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

func is_str_set(m map[string]interface{}, key string) bool {
	return m[key] != nil && len(m[key].([]string)) > 0
}


// this method check if a field exists in a map
func get_fields_if_exist(m map[string]interface{}, fields ...string) (interface{}, bool) {
	var temp_map map[string]interface{}
	temp_map = m
	for i, field := range fields {
		switch temp_map[field].(type) {
		case map[string]interface{}:
			temp_map = temp_map[field].(map[string]interface{})
		case nil:
			return nil, false
		default:
			// if it is the last field and it is not empty
			// it exists othewise return false
			if i == len(fields) - 1 {
				return temp_map[field], true
			}
			return nil, false
		}
	}
	return temp_map, true
}

func temp_file_in_work_dir(destination string) string {
	dst_path := path.Join(destination, _TEMP_NAME)
	if _, err := os.Stat(dst_path); os.IsNotExist(err) {
		os.Mkdir(dst_path, 0777)
	}
	return temp_file(dst_path)
}

func temp_file(destination string) string {
	temp_dst, err := ioutil.TempFile(destination, _TEMP_PREFIX)
	if err == nil {
		return temp_dst.Name()
	}
	return ""
}

func download_from_url(rawurl string, filepath string) error {
	file, err := os.Open(filepath)
	if err == nil {
		response, err1 := http.Get(rawurl)
		if err1 != nil {
			logrus.Error(fmt.Sprintf("Error while downloading error: %s", err1))
			return err1
		}
		defer response.Body.Close()
		n, ok := io.Copy(file, response.Body)
		if ok != nil {
			logrus.Error(fmt.Sprintf("Error while copying file: %s", ok))
			return ok
		}
		logrus.Info(fmt.Sprintf("%s bytes downloaded successfully", n))
		return nil
	}
	return err
}

func Get_response_data(event *events.Event, event_data map[string]interface{}) *events.Event {
	// TODO not implemented
	/*
	resource_type := event.ResourceType
	var ihm model.InstanceHostMap
	mapstructure.Decode(event_data, &ihm)
	tp := ihm.Type
	if tp != nil && len(tp) > 0{
		r := regexp.Compile("([A-Z])")
		inner_name := strings.Replace(tp, r.FindStringSubmatch(tp)[0], "_\1", -1)
		method_name := strings.ToLower(fmt.Sprintf("_get_%s_data", inner_name))
		method := ""

	}
	*/
	return &events.Event{}
}