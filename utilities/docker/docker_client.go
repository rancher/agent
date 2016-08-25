package docker

import (
	"fmt"
	"github.com/docker/engine-api/client"
	"github.com/rancher/agent/utilities/config"
	"github.com/rancher/agent/utilities/constants"
	"golang.org/x/net/context"
	"os"
)

func GetClient(version string, timeout string) *client.Client {
	if config.UseBoot2dockerConnectionEnvVars() {
		return launchDefaultCli(version)
	}
	defCli := launchDefaultCli(version)
	if defCli != nil {
		return defCli
	}
	cli, err := client.NewClient(fmt.Sprintf("tcp://%v:2375", os.Getenv("CATTLE_AGENT_IP")), version, nil, map[string]string{
		"timeout": timeout,
	})
	if err != nil {
		panic(err)
	}
	return cli
}

func launchDefaultCli(version string) *client.Client {
	cli, err := client.NewEnvClient()
	if err != nil {
		return nil
	}
	cli.UpdateClientVersion(version)
	return cli
}

var DefaultClient = GetClient(constants.DefaultVersion, "0")

var info, err = DefaultClient.Info(context.Background())
var Info = info
var InfoErr = err
