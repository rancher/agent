// +build linux freebsd solaris openbsd darwin

package docker

import (
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	dclient "github.com/docker/docker/client"
	"github.com/docker/go-connections/tlsconfig"
	"github.com/pkg/errors"
	"github.com/rancher/agent/utils/constants"
)

var (
	versionClients = map[string]*dclient.Client{}
	timeoutClients = map[time.Duration]*dclient.Client{}
	clientLock     = sync.Mutex{}
)

const DefaultVersion = "1.22"

func launchDefaultClient(version string) (*dclient.Client, error) {
	clientLock.Lock()
	defer clientLock.Unlock()

	if c, ok := versionClients[version]; ok {
		return c, nil
	}

	cli, err := dclient.NewEnvClient()
	if err != nil {
		return nil, errors.Wrap(err, constants.LaunchDefaultClientError)
	}
	cli.UpdateClientVersion(version)

	versionClients[version] = cli
	return cli, nil
}

func NewEnvClientWithTimeout(timeout time.Duration) (*dclient.Client, error) {
	clientLock.Lock()
	defer clientLock.Unlock()

	if c, ok := timeoutClients[timeout]; ok {
		return c, nil
	}

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

	host := os.Getenv("DOCKER_HOST")
	if host == "" {
		host = dclient.DefaultDockerHost
	}

	version := os.Getenv("DOCKER_API_VERSION")
	if version == "" {
		version = dclient.DefaultVersion
	}

	c, err := dclient.NewClient(host, version, client, nil)
	if err != nil {
		return nil, err
	}

	timeoutClients[timeout] = c
	return c, nil
}
