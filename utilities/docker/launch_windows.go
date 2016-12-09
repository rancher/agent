package docker

import (
	"fmt"
	dockerClient "github.com/docker/docker/client"
	"github.com/docker/go-connections/tlsconfig"
	"github.com/pkg/errors"
	"github.com/rancher/agent/utilities/constants"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const DefaultVersion = "1.24"

func launchDefaultClient(version string) (*dockerClient.Client, error) {
	ip := fmt.Sprintf("tcp://%v:2375", os.Getenv("DEFAULT_GATEWAY"))
	if os.Getenv("DEFAULT_GATEWAY") == "" {
		client, err := dockerClient.NewEnvClient()
		if err != nil {
			return nil, err
		}
		client.UpdateClientVersion(DefaultVersion)
		return client, nil
	}
	cliFromAgent, cerr := dockerClient.NewClient(ip, version, nil, nil)
	if cerr != nil {
		return nil, errors.Wrap(cerr, constants.LaunchDefaultClientError)
	}
	return cliFromAgent, nil
}

func NewEnvClientWithTimeout(timeout time.Duration) (*dockerClient.Client, error) {
	var client *http.Client
	if dockerCertPath := os.Getenv("DOCKER_CERT_PATH"); dockerCertPath != "" {
		options := tlsconfig.Options{
			CAFile:             filepath.Join(dockerCertPath, "ca.pem"),
			CertFile:           filepath.Join(dockerCertPath, "cert.pem"),
			KeyFile:            filepath.Join(dockerCertPath, "key.pem"),
			InsecureSkipVerify: os.Getenv("DOCKER_TLS_VERIFY") == "",
		}
		tlsc, err := tlsconfig.Client(options)
		if err != nil {
			return nil, err
		}

		client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: tlsc,
			},
			Timeout: timeout,
		}
	}

	host := fmt.Sprintf("tcp://%v:2375", os.Getenv("DEFAULT_GATEWAY"))
	if os.Getenv("DEFAULT_GATEWAY") == "" {
		host = dockerClient.DefaultDockerHost
	}

	version := os.Getenv("DOCKER_API_VERSION")
	if version == "" {
		version = dockerClient.DefaultVersion
	}

	return dockerClient.NewClient(host, version, client, nil)
}
