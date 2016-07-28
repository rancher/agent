package hostInfo

import (
	"encoding/json"
	"github.com/Sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"time"
)

type CadvisorAPIClient struct {
	URL string
}

func (c CadvisorAPIClient) GetContainers() (map[string]interface{}, error) {
	return c.get(c.URL + "/containers")
}

func (c CadvisorAPIClient) GetLatestStat() map[string]interface{} {
	containers := c.GetStats()
	if len(containers) > 1 {
		return containers[len(containers)-1].(map[string]interface{})
	}
	return map[string]interface{}{}
}

func (c CadvisorAPIClient) GetStats() []interface{} {
	containers, err := c.GetContainers()
	if err != nil {
		logrus.Error(err)
	}
	if len(containers) > 0 {
		return containers["stats"].([]interface{})
	}
	return []interface{}{}
}

func (c CadvisorAPIClient) get(url string) (map[string]interface{}, error) {
	resp, err := http.Get(url)
	if err == nil {
		defer resp.Body.Close()
		data, _ := ioutil.ReadAll(resp.Body)
		var result map[string]interface{}
		err1 := json.Unmarshal(data, result)
		if err1 != nil {
			logrus.Error(err1)
		}
		return result, nil
	}
	return map[string]interface{}{}, err
}

func (c CadvisorAPIClient) GetMachineStats() map[string]interface{} {
	machineData, err := c.get(c.URL + "/machine")
	if err != nil {
		logrus.Error(err)
	}
	if len(machineData) > 0 {
		return machineData
	}
	return map[string]interface{}{}
}

func (c CadvisorAPIClient) TimestampDiff(timeCurrent, timePrev string) int64 {
	timeCurConv, _ := time.Parse(time.RFC3339, timeCurrent[0:26])
	timePrevConv, _ := time.Parse(time.RFC3339, timePrev[0:26])
	diff := timeCurConv.Sub(timePrevConv)
	return int64(diff)
}
