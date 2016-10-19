// +build linux freebsd solaris openbsd darwin

package config

import (
	gofqdn "github.com/ShowMax/go-fqdn"
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
		// if command line doesn't work, try to look up by IP
		fqdn := gofqdn.Get()
		return fqdn, nil
	}
	fqdn := string(output)
	fqdn = fqdn[:len(fqdn)-1]
	return fqdn, nil
}
