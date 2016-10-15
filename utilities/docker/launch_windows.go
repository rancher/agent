package docker

import (
	"fmt"
	"github.com/docker/engine-api/client"
	dclient "github.com/docker/engine-api/client"
	"github.com/docker/go-connections/tlsconfig"
	"github.com/pkg/errors"
	"github.com/rancher/agent/utilities/constants"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func launchDefaultClient(version string) (*client.Client, error) {
	ip := fmt.Sprintf("tcp://%v:2375", os.Getenv("DEFAULT_GATEWAY"))
	cliFromAgent, cerr := client.NewClient(ip, version, nil, nil)
	if cerr != nil {
		return nil, errors.Wrap(cerr, constants.LaunchDefaultClientError)
	}
	return cliFromAgent, nil
}

func NewEnvClientWithTimeout(timeout time.Duration) (*dclient.Client, error) {
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

	version := os.Getenv("DOCKER_API_VERSION")
	if version == "" {
		version = dclient.DefaultVersion
	}

	return dclient.NewClient(host, version, client, nil)
}
