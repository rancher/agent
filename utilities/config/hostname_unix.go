package config

import (
	"github.com/pkg/errors"
	"github.com/rancher/agent/utilities/constants"
	"os/exec"
)

func Hostname() (string, error) {
	hostname, err := getFQDNLinux()
	if err != nil {
		return "", errors.Wrap(err, constants.HostNameError)
	}
	return DefaultValue("HOSTNAME", hostname), nil
}

func getFQDNLinux() (string, error) {
	cmd := exec.Command("/bin/hostname", "-f")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.Wrap(err, constants.GetFQDNLinuxError)
	}
	fqdn := string(output)
	fqdn = fqdn[:len(fqdn)-1]
	return fqdn, nil
}
