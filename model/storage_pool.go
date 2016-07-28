package model

type StoragePool struct {
	AgentID            int         `json:"agentId"`
	Created            int64       `json:"created"`
	Description        interface{} `json:"description"`
	External           bool        `json:"external"`
	ID                 int         `json:"id"`
	Kind               string      `json:"kind"`
	Name               interface{} `json:"name"`
	PhysicalTotalBytes interface{} `json:"physicalTotalBytes"`
	PhysicalUsedBytes  interface{} `json:"physicalUsedBytes"`
	Removed            interface{} `json:"removed"`
	State              string      `json:"state"`
	Type               string      `json:"type"`
	UUID               string      `json:"uuid"`
	VirtualTotalBytes  interface{} `json:"virtualTotalBytes"`
}
