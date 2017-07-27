// +build linux freebsd solaris openbsd

package utils

import "syscall"

var SysAttr = &syscall.SysProcAttr{
	Setpgid:   true,
	Pdeathsig: syscall.SIGKILL,
}
