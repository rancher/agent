// +build linux freebsd solaris openbsd darwin

package utils

import (
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	dclient "github.com/docker/docker/client"
	"github.com/docker/go-connections/tlsconfig"
)

var (
	versionClients = map[string]*dclient.Client{}
	timeoutClients = map[time.Duration]*dclient.Client{}
	clientLock     = sync.Mutex{}
)

const (
	DefaultVersion    = "1.22"
	DockerRuntime     = "docker"
	ContainerdRuntime = "containerd"
)

func launchDefaultClient(version string) (*dclient.Client, error) {
	clientLock.Lock()
	defer clientLock.Unlock()

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
			Timeout: time.Second * 30,
		}
	}

	host := os.Getenv("DOCKER_HOST")
	if host == "" {
		host = dclient.DefaultDockerHost
	}

	ver := os.Getenv("DOCKER_API_VERSION")
	if ver == "" {
		ver = version
	}

	c, err := dclient.NewClient(host, ver, client, nil)
	if err != nil {
		return nil, err
	}

	return c, nil
}
