//+build !windows

package utils

import (
	"github.com/docker/engine-api/types"
)

func getIP(inspect types.ContainerJSON) string {
	return inspect.NetworkSettings.IPAddress
}
