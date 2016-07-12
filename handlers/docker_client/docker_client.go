package docker_client

import (
	"os"
	"github.com/docker/engine-api/client"
	"github.com/Sirupsen/logrus"
)

func GetClient(version string) (*client.Client){
// Launch client from environment variables if go-agent is not running on host
	env_err := os.Setenv("DOCKER_API_VERSION", version)
	if env_err != nil {
		logrus.Error(env_err)
	}
	cli, err := client.NewEnvClient()
	if err != nil {
		logrus.Error(env_err)
	}
	return cli
}