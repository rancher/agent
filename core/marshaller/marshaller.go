package marshaller

import (
	"encoding/json"

	"github.com/leodotcloud/log"
	"github.com/pkg/errors"
	"github.com/rancher/agent/utilities/constants"
)

func FromString(rawstring string) map[string]interface{} {
	obj := map[string]interface{}{}
	err := json.Unmarshal([]byte(rawstring), &obj)
	if err != nil {
		log.Error(err)
	}
	return obj
}

func StructToMap(v interface{}) (map[string]interface{}, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return map[string]interface{}{}, errors.Wrap(err, constants.StructToMapError+"failed to marshal data")
	}
	event := map[string]interface{}{}
	if err := json.Unmarshal(b, &event); err != nil {
		return map[string]interface{}{}, errors.Wrap(err, constants.StructToMapError+"failed to unmarshal data")
	}
	return event, nil
}
