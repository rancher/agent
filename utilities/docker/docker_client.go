package docker

import (
	"github.com/docker/engine-api/client"
	"github.com/rancher/agent/utilities/constants"
	"golang.org/x/net/context"
)

func GetClient(version string) *client.Client {
	defCli, err := launchDefaultClient(version)
	if err != nil {
		panic(err)
	}
	return defCli
}

func launchDefaultClient(version string) (*client.Client, error) {
	cli, err := client.NewEnvClient()
	if err != nil {
		return nil, err
	}
	cli.UpdateClientVersion(version)
	return cli, nil
}

var DefaultClient = GetClient(constants.DefaultVersion)

var info, err = DefaultClient.Info(context.Background())
var Info = info
var InfoErr = err
