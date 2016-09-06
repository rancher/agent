package config

import "github.com/pkg/errors"

func Hostname() (string, error) {

	hostname, err = os.Hostname()
	if err != nil {
		return "", errors.Wrap(err)
	}
	return DefaultValue("HOSTNAME", hostname)
}
