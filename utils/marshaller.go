package utils

import (
	"encoding/json"

	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
)

func FromString(rawstring string) map[string]interface{} {
	obj := map[string]interface{}{}
	err := json.Unmarshal([]byte(rawstring), &obj)
	if err != nil {
		logrus.Error(err)
	}
	return obj
}

func StructToMap(v interface{}) (map[string]interface{}, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return map[string]interface{}{}, errors.Wrap(err, "failed to marshal data")
	}
	event := map[string]interface{}{}
	if err := json.Unmarshal(b, &event); err != nil {
		return map[string]interface{}{}, errors.Wrap(err, "failed to unmarshal data")
	}
	return event, nil
}

func Unmarshalling(data interface{}, v interface{}) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return errors.Wrap(err, "failed to marshall object")
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return errors.Wrap(err, "failed to unmarshall object")
	}
	return nil
}