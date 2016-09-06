package config

import (
	"github.com/pkg/errors"
	"os"
	"github.com/rancher/agent/utilities/constants"
)

func Hostname() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", errors.Wrap(err, constants.HostNameError)
	}
	return DefaultValue("HOSTNAME", hostname), nil
}
