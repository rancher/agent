package utils

import "encoding/json"

func ConvertByJSON(src, target interface{}) error {
	bytes, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, target)
}
