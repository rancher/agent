// +build linux freebsd solaris openbsd darwin

package docker

import (
	"github.com/docker/engine-api/client"
	"github.com/pkg/errors"
)

func launchDefaultClient(version string) (*client.Client, error) {
	cli, err := client.NewEnvClient()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	cli.UpdateClientVersion(version)
	return cli, nil
}
