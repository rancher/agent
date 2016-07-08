package handlers

import (
	"strings"
	"os"
	"fmt"
	"encoding/json"
	"errors"
	"io"
	"reflect"
)

func unwrap(obj map[string]interface{}) map[string]interface {


}

func add_label(config *map[string]interface{}, new_labels map[string]string){
	labels, ok := config["labels"]
	if ok != nil {
		labels = make(map[string]string)
		config["labels"] = labels
	}
	update(config, new_labels)
}

func update(config *map[string]interface{}, new_labels map[string]string){
	for key, value := range new_labels {
		config["labels"].(map[string]string)[key] = value
	}
}

func search_in_list(slice []string, target string) bool {
	for _, value := range slice {
		if strings.Compare(target, value) {
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

func is_nonrancher_container(instance Instance) bool {
	return instance.NativeContainer
}

func add_to_env(config *map[string]interface{}, result *map[string]string, args ...string){
	if env, ok := config["enviroment"]; !ok {
		env = make(map[string]string)
		config["enviroment"] = env
	} else {
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

func get_or_create_port_list(config *map[string]interface{}, key string) *[]Port {
	list, ok := config[key]
	if !ok {
		list = list.([]Port{})
		config[key] = list
	}

	return &config[key]
}

func get_or_create_binding_map(config *map[string]interface{}, key string) *map[string]string {
	m, ok := config[key]
	if !ok {
		m = make(map[string]string)
		config[key] = m
	}
	return &config[key]
}

func has_key(m map[string]interface{}, key string) bool {
	_, ok := m[key]
	return ok
}

//TODO implement this function
func check_output(kwargs map[string]interface{}, args ...string){

}

func has_label(instance *Instance){
	return instance.Labels["io.rancher.container.cattle_url"]
}

func readBuffer(reader io.ReadCloser) string {
	buffer := [1024]byte{}
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
	return m[key] != nil && len(m[key]) > 0
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