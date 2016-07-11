package model

import (
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/container"
)

type Tuple struct {
	Src, Dest string
}

type VolumeCreateRequest types.VolumeCreateRequest

type Host_Config container.HostConfig

type Option_Config struct {
	Key string
	Dev_List []map[string]string
	Docker_Field string
	Field string
}

type Image_Params struct {
	Image Image
	Tag string
	Mode string
	Complete bool
}

type Config container.Config

type AuthConfig types.AuthConfig

type Method func(map[string]string, []string) (interface{}, error)
