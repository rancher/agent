package model

type Host struct {
	AgentID int `json:"agentId"`
	Created int64 `json:"created"`
	Data map[string]interface{}
	Description interface{} `json:"description"`
	ID int `json:"id"`
	Kind string `json:"kind"`
	Name interface{} `json:"name"`
	Removed interface{} `json:"removed"`
	State string `json:"state"`
	Type string `json:"type"`
	UUID string `json:"uuid"`
}

