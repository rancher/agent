package hostInfo

import (
	"encoding/json"
	"github.com/rancher/agent/utilities/config"
	"io/ioutil"
	"os"
	"path"
)

type CloudProviderCollector struct{}

type Info struct {
	Labels map[string]string `json:"label,omitempty"`
}

func (c CloudProviderCollector) GetData() (map[string]interface{}, error) {
	return nil, nil
}

func (c CloudProviderCollector) KeyName() string {
	return "cloudProvider"
}

func (c CloudProviderCollector) GetLabels(prefix string) (map[string]string, error) {
	file, err := os.Open(path.Join(config.StateDir(), "info.json"))
	// if file doesn't exit, just skip it
	if err != nil {
		return nil, nil
	}
	defer file.Close()
	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}
	var i Info
	err = json.Unmarshal(bytes, &i)
	if err != nil {
		return nil, err
	}
	return i.Labels, nil
}
