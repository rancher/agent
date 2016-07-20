package model

type InstancePull struct {
	Kind     string `json:"kind"`
	Image    Image
	Mode     string `json:"mode"`
	Complete bool   `json:"complete"`
	Tag      string `json:"tag"`
}
