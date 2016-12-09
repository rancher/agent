package events

import (
	"fmt"
	"github.com/docker/docker/client"
	"os"
)

const (
	defaultApiVersion = "1.24"
)

func NewDockerClient() (*client.Client, error) {
	host := fmt.Sprintf("tcp://%v:2375", os.Getenv("DEFAULT_GATEWAY"))
	if os.Getenv("DEFAULT_GATEWAY") == "" {
		client, err := client.NewEnvClient()
		if err != nil {
			return nil, err
		}
		client.UpdateClientVersion(defaultApiVersion)
		return client, nil
	}
	return client.NewClient(host, defaultApiVersion, nil, nil)
}
