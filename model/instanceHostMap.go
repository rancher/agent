package model

type InstanceHostMap struct {
	Host       Host
	HostID     int `json:"hostId"`
	ID         int `json:"id"`
	Instance   Instance
	InstanceID int         `json:"instanceId"`
	Removed    interface{} `json:"removed"`
	State      string      `json:"state"`
	Type       string      `json:"type"`
}
