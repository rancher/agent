// +build linux freebsd solaris openbsd darwin

package utils

import (
	"os/exec"

	gofqdn "github.com/ShowMax/go-fqdn"
)

func Hostname() (string, error) {
	hostname, err := getFQDNLinux()
	if err != nil {
		return "", err
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
