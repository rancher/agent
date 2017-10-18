package hostInfo

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"

<<<<<<< HEAD:host_info/cloud_provider.go
	"github.com/rancher/agent/utils"
=======
	"github.com/rancher/agent/utilities/config"
>>>>>>> b2575c6... add flag to control cloud provider:core/hostInfo/cloud_provider.go
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
<<<<<<< HEAD:host_info/cloud_provider.go
	file, err := os.Open(path.Join(utils.StateDir(), "info.json"))
=======
	if !config.DetectCloudProvider() {
		return nil, nil
	}
	file, err := os.Open(path.Join(config.StateDir(), "info.json"))
>>>>>>> b2575c6... add flag to control cloud provider:core/hostInfo/cloud_provider.go
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
