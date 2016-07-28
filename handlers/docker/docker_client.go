package docker

import (
	"github.com/Sirupsen/logrus"
	"github.com/docker/engine-api/client"
	"os"
)

func GetClient(version string) *client.Client {
	// Launch client from environment variables if go-agent is not running on host
	envErr := os.Setenv("DOCKER_API_VERSION", version)
	if envErr != nil {
		logrus.Error(envErr)
	}
	cli, err := client.NewEnvClient()
	if err != nil {
		logrus.Error(err)
	}
	return cli
}
