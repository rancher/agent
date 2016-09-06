package hostInfo

import (
	"github.com/pkg/errors"
	"github.com/rancher/agent/utilities/constants"
	"math"
	"strconv"
	"strings"
)

func (m MemoryCollector) parseMemInfo() (map[string]interface{}, error) {
	data := map[string]interface{}{}
	memData, err := m.DataGetter.GetMemInfoData()
	if err != nil {
		return data, errors.Wrap(err, constants.ParseMemInfoError)
	}
	for _, line := range memData {
		lineList := strings.Split(line, ":")
		keyLower := strings.ToLower(lineList[0])
		possibleMemValue := strings.Split(strings.TrimSpace(lineList[1]), " ")[0]

		if _, ok := KeyMap[keyLower]; ok {
			value, _ := strconv.ParseFloat(possibleMemValue, 64)
			convertMemVal := value / m.Unit
			convertMemVal = math.Floor(convertMemVal*1000) / 1000
			data[KeyMap[keyLower]] = convertMemVal
		}
	}
	return data, nil
}
