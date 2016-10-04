// +build linux freebsd solaris openbsd darwin

package hostInfo

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/pkg/errors"
	"github.com/rancher/agent/utilities/utils"
)

func (i IopsCollector) getIopsData(readOrWrite string) (map[string]interface{}, error) {
	file, err := os.Open("/var/lib/rancher/state/" + readOrWrite + ".json")
	if err != nil {
		return map[string]interface{}{}, err
	}
	data, _ := ioutil.ReadAll(file)
	var result map[string]interface{}
	json.Unmarshal(data, result)
	return result, nil
}

func (i IopsCollector) parseIopsData() (map[string]interface{}, error) {
	data := map[string]interface{}{}
	readJSONData, err := i.getIopsData("read")
	if err != nil && !os.IsNotExist(err) {
		return data, errors.WithStack(err)
	} else if err != nil && os.IsNotExist(err) {
		return data, nil
	}
	writeJSONData, err := i.getIopsData("write")
	if err != nil && !os.IsNotExist(err) {
		return data, errors.WithStack(err)
	} else if err != nil && os.IsNotExist(err) {
		return data, nil
	}
	readIops, _ := utils.GetFieldsIfExist(readJSONData["jobs"].([]map[string]interface{})[0], "read", "iops")
	writeIops, _ := utils.GetFieldsIfExist(writeJSONData["jobs"].([]map[string]interface{})[0], "write", "iops")
	device, _ := utils.GetFieldsIfExist(readJSONData["disk_util"].([]map[string]interface{})[0], "name")
	key := "/dev" + strconv.QuoteToASCII(device.(string))
	data[key] = map[string]interface{}{
		"read":  readIops,
		"write": writeIops,
	}
	return data, nil
}

func (i IopsCollector) getDefaultDisk() (string, error) {
	data, err := i.GetData()
	if err != nil {
		return "", errors.WithStack(err)
	}
	if len(data) == 0 {
		return "", nil
	}
	for key := range data {
		return key, nil
	}
	return "", nil
}
