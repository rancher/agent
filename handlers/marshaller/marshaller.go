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
