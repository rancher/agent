package utils

import (
	"github.com/docker/docker/client"
)

func GetRuntimeClient(runtime, version string) *client.Client {
	if runtime == DockerRuntime {
		defCli, err := launchDefaultClient(version)
		if err != nil {
			panic(err)
		}
		return defCli
	}
	return nil
}
