package marshaller

import (
	"encoding/json"

	"github.com/pkg/errors"
)

func FromString(rawstring string) (map[string]interface{}, error) {
	obj := map[string]interface{}{}
	err := json.Unmarshal([]byte(rawstring), &obj)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return obj, nil
}

func ToString(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func StructToMap(v interface{}) (map[string]interface{}, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return map[string]interface{}{}, errors.WithStack(err)
	}
	event := map[string]interface{}{}
	if err := json.Unmarshal(b, &event); err != nil {
		return map[string]interface{}{}, errors.WithStack(err)
	}
	return event, nil
}
