package hostInfo

import (
	"github.com/Sirupsen/logrus"
	"github.com/rancher/agent/utilities/utils"
	"time"
)

type CadvisorGetter interface {
	GetContainers() (map[string]interface{}, error)
	GetMachineStats() (map[string]interface{}, error)
	Get(string) (map[string]interface{}, error)
}

type CadvisorAPIClient struct {
	dataGetter CadvisorGetter
}

type CadvisorDataGetter struct {
	URL string
}

func (c CadvisorDataGetter) GetContainers() (map[string]interface{}, error) {
	return c.Get(c.URL + "/containers")
}

func (c CadvisorAPIClient) GetLatestStat() map[string]interface{} {
	containers := c.GetStats()
	if len(containers) > 1 {
		return containers[len(containers)-1].(map[string]interface{})
	}
	return map[string]interface{}{}
}

func (c CadvisorAPIClient) GetStats() []interface{} {
	containers, err := c.dataGetter.GetContainers()
	if err != nil {
		logrus.Error(err)
	}
	if len(containers) > 0 {
		return utils.InterfaceToArray(containers["stats"])
	}
	return []interface{}{}
}

func (c CadvisorDataGetter) GetMachineStats() (map[string]interface{}, error) {
	machineData, err := c.Get(c.URL + "/machine")
	if err != nil {
		return nil, err
	}
	return machineData, nil
}

func (c CadvisorAPIClient) TimestampDiff(timeCurrent, timePrev string) float64 {
	timeCurConv, _ := time.Parse(time.RFC3339, timeCurrent[0:26])
	timePrevConv, _ := time.Parse(time.RFC3339, timePrev[0:26])
	diff := timeCurConv.Sub(timePrevConv)
	return float64(diff)
}

func (c CadvisorDataGetter) Get(url string) (map[string]interface{}, error) {
	return utils.Get(url)
}
