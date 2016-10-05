// +build linux freebsd solaris openbsd darwin

package config

import (
	"os/exec"

	"github.com/pkg/errors"
)

func Hostname() (string, error) {
	hostname, err := getFQDNLinux()
	if err != nil {
		return "", errors.WithStack(err)
	}
	return DefaultValue("HOSTNAME", hostname), nil
}

func getFQDNLinux() (string, error) {
	cmd := exec.Command("/bin/hostname", "-f")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.WithStack(err)
	}
	fqdn := string(output)
	fqdn = fqdn[:len(fqdn)-1]
	return fqdn, nil
}
