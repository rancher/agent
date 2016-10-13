package docker

import (
	"fmt"
	"github.com/docker/engine-api/client"
	"github.com/pkg/errors"
	"github.com/rancher/agent/utilities/constants"
)

func launchDefaultClient(version string) (*client.Client, error) {
	ip := fmt.Sprintf("tcp://%v:2375", "172.16.0.1")
	cliFromAgent, cerr := client.NewClient(ip, version, nil, nil)
	if cerr != nil {
		return nil, errors.Wrap(cerr, constants.LaunchDefaultClientError)
	}
	return cliFromAgent, nil
}
