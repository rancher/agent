package marshaller

import (
	"encoding/json"
	"errors"
)

func UnmarshalEventList(rawEvent []byte) []map[string]interface{} {
	events := []map[string]interface{}{}
	err := json.Unmarshal(rawEvent, &events)
	if err != nil {
		panic(errors.New("Error unmarshalling event %v"))
	}
	return events
}

func FromString(rawstring string) map[string]interface{} {
	obj := map[string]interface{}{}
	err := json.Unmarshal([]byte(rawstring), &obj)
	if err != nil {
		panic(errors.New("Error unmarshalling event %v"))
	}
	return obj
}

func ToString(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}
