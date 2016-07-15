package marshaller

import (
	"encoding/json"
	"github.com/Sirupsen/logrus"
)

func FromString(rawstring string) map[string]interface{} {
	obj := map[string]interface{}{}
	err := json.Unmarshal([]byte(rawstring), &obj)
	if err != nil {
		logrus.Error(err)
	}
	return obj
}

func ToString(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func ToMap(v interface{}) map[string]interface{} {
	rawByte, _ := json.Marshal(v)
	event := map[string]interface{}{}
	err := json.Unmarshal(rawByte, &event)
	if err != nil {
		logrus.Infof("Error unmarshalling event %v", err)
	}
	return event
}
