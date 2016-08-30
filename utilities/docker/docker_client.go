package docker

import (
	"fmt"
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/rancher/agent/utilities/constants"
	"golang.org/x/net/context"
	"os"
	"runtime"
	"time"
)

func GetClient(version string) *client.Client {
	defCli, err := launchDefaultClient(version)
	if err != nil {
		panic(err)
	}
	return defCli
}

func launchDefaultClient(version string) (*client.Client, error) {
	if runtime.GOOS == "windows" {
		ip := fmt.Sprintf("tcp://%v:2375", os.Getenv("CATTLE_AGENT_IP"))
		cliFromAgent, cerr := client.NewClient(ip, version, nil, nil)
		if cerr != nil {
			return nil, cerr
		}
		return cliFromAgent, nil
	}
	cli, err := client.NewEnvClient()
	if err != nil {
		return nil, err
	}
	cli.UpdateClientVersion(version)
	return cli, nil
}

func init() {
	// try 10 times to see if we could get the Info from docker daemon
	for i := 0; i < 10; i++ {
		info, err := DefaultClient.Info(context.Background())
		if err == nil {
			Info = info
			break
		}
		time.Sleep(time.Duration(1) * time.Second)
	}
}

var DefaultClient = GetClient(constants.DefaultVersion)

var Info types.Info
