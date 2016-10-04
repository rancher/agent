package docker

import (
	"fmt"
	"os"

	"github.com/docker/engine-api/client"
	"github.com/pkg/errors"
)

func launchDefaultClient(version string) (*client.Client, error) {
	ip := fmt.Sprintf("tcp://%v:2375", os.Getenv("CATTLE_AGENT_IP"))
	cliFromAgent, cerr := client.NewClient(ip, version, nil, nil)
	if cerr != nil {
		return nil, errors.WithStack(err)
	}
	return cliFromAgent, nil
}
