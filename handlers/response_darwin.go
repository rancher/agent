package handlers

import (
	"github.com/docker/docker/api/types"
	"github.com/rancher/agent/progress"
)

func getIP(inspect types.ContainerJSON, networkMode string, pro *progress.Progress) (string, error) {
	return "", nil
}
