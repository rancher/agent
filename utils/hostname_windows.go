package utils

import (
	"github.com/pkg/errors"
	"github.com/rancher/agent/utils/constants"
	"os"
)

func Hostname() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", errors.Wrap(err, constants.HostNameError)
	}
	return DefaultValue("HOSTNAME", hostname), nil
}
