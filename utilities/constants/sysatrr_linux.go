// +build linux freebsd solaris openbsd darwin
// TODO: pdeath is linux only.  so you need !linux version

package constants

import "syscall"

var SysAttr = &syscall.SysProcAttr{
	Setpgid:   true,
	Pdeathsig: syscall.SIGKILL,
}
