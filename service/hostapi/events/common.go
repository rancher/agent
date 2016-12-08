package events

import (
	"github.com/docker/docker/api/types"
	"golang.org/x/net/context"
)

type SimpleDockerClient interface {
	ContainerInspect(context context.Context, id string) (types.ContainerJSON, error)
}
