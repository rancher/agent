package docker

import (
	"github.com/Sirupsen/logrus"
	"github.com/docker/engine-api/client"
	"net/http"
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

func GetClientFromUrl(host string, version string, httpClient *http.Client, headers map[string]string) *client.Client {
	cli, err := client.NewClient(host, version, httpClient, headers)
	if err != nil {
		return nil
	}
	return cli
}

//var DefaultClient = GetClient(constants.DefaultVersion)
var DefaultClient = GetClientFromUrl("tcp://192.168.42.177:2375", "v1.24", nil, nil)
