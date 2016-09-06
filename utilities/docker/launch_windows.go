package docker

import (
	"github.com/pkg/errors"
	"github.com/rancher/agent/utilities/constants"
	"github.com/docker/engine-api/client"
	"fmt"
	"os"
)

func launchDefaultClient(version string) (*client.Client, error) {
	ip := fmt.Sprintf("tcp://%v:2375", os.Getenv("CATTLE_AGENT_IP"))
	cliFromAgent, cerr := client.NewClient(ip, version, nil, nil)
	if cerr != nil {
		return nil, errors.Wrap(cerr, constants.LaunchDefaultClientError)
	}
	return cliFromAgent, nil
}
