package model

type Event struct {
	Data map[string]interface{}
	ID string `json:"id"`
	Name string `json:"name"`
	PreviousIds interface{} `json:"previousIds"`
	PreviousNames interface{} `json:"previousNames"`
	Publisher interface{} `json:"publisher"`
	ReplyTo string `json:"replyTo"`
	ResourceID string `json:"resourceId"`
	ResourceType string `json:"resourceType"`
	Time int64 `json:"time"`
}
