package model

import "github.com/docker/docker/api/types"

type InfoData struct {
	Info    types.Info
	Version types.Version
}
