//+build !windows

package utils

import (
	"github.com/docker/docker/api/types"
)

func getIP(inspect types.ContainerJSON) string {
	return inspect.NetworkSettings.IPAddress
}
