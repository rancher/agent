package docker

import (
	"github.com/docker/engine-api/client"
	"github.com/pkg/errors"
	"github.com/rancher/agent/utilities/constants"
)

func launchDefaultClient(version string) (*client.Client, error) {
	cli, err := client.NewEnvClient()
	if err != nil {
		return nil, errors.Wrap(err, constants.LaunchDefaultClientError)
	}
	cli.UpdateClientVersion(version)
	return cli, nil
}
