package hostInfo

import (
	"encoding/json"
	"github.com/Sirupsen/logrus"
	"github.com/rancher/agent/utilities/utils"
	"io/ioutil"
	"os"
	"strconv"
)

type IopsCollector struct {
	GOOS string
}

func (i IopsCollector) getIopsData(readOrWrite string) (map[string]interface{}, error) {
	file, err := os.Open("/var/lib/rancher/state/" + readOrWrite + ".json")
	if err != nil {
		logrus.Error(err)
		return map[string]interface{}{}, err
	}
	data, _ := ioutil.ReadAll(file)
	var result map[string]interface{}
	json.Unmarshal(data, result)
	return result, nil
}

func (i IopsCollector) parseIopsData() map[string]interface{} {
	data := map[string]interface{}{}
	readJSONData, err1 := i.getIopsData("read")
	writeJSONData, err2 := i.getIopsData("write")
	if err1 != nil || err2 != nil {
		return data
	}
	readIops, _ := utils.GetFieldsIfExist(readJSONData["jobs"].([]map[string]interface{})[0], "read", "iops")
	writeIops, _ := utils.GetFieldsIfExist(writeJSONData["jobs"].([]map[string]interface{})[0], "write", "iops")
	device, _ := utils.GetFieldsIfExist(readJSONData["disk_util"].([]map[string]interface{})[0], "name")
	key := "/dev" + strconv.QuoteToASCII(device.(string))
	data[key] = map[string]interface{}{
		"read":  readIops,
		"write": writeIops,
	}
	return data
}

func (i IopsCollector) GetData() map[string]interface{} {
	if i.GOOS == "linux" {
		return i.parseIopsData()
	}
	return map[string]interface{}{}
}

func (i IopsCollector) getDefaultDisk() string {
	data := i.GetData()
	if len(data) == 0 {
		return ""
	}
	for key := range data {
		return key
	}
	return ""
}

func (i IopsCollector) KeyName() string {
	return "iopsInfo"
}

func (i IopsCollector) GetLabels(prefix string) map[string]string {
	return map[string]string{}
}
