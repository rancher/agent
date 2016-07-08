package marshaller

import (
	"encoding/json"
	"errors"
)

//TODO unmarshal byte array into a map. May require to unmarshal a json array
func UnmarshalEventList(rawEvent []byte) map[string]interface{} {
	events := []map[string]interface{}{}
	err := json.Unmarshal(rawEvent, &events)
	if err != nil {
		panic(errors.New("Error unmarshalling event %v"))
	}
	return events
}

func From_string(rawstring string) map[string]interface{} {

}
