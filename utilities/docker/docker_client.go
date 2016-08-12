package docker

import (
	"github.com/Sirupsen/logrus"
	"github.com/docker/engine-api/client"
	"github.com/rancher/agent/utilities/constants"
)

func GetClient(version string) *client.Client {
	// Launch client from environment variables if go-agent is not running on host
	cli, err := client.NewEnvClient()
	if err != nil {
		logrus.Error(err)
	}
	cli.UpdateClientVersion(version)
	return cli
}

var DefaultClient = GetClient(constants.DefaultVersion)
