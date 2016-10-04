package config

import (
	"os"

	"github.com/pkg/errors"
)

func Hostname() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", errors.WithStack(err)
	}
	return DefaultValue("HOSTNAME", hostname), nil
}
