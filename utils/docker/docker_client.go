package docker

import (
	"github.com/docker/docker/client"
)

func GetClient(version string) *client.Client {
	defCli, err := launchDefaultClient(version)
	if err != nil {
		panic(err)
	}
	return defCli
}
