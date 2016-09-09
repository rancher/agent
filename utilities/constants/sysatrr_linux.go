// +build linux freebsd solaris openbsd darwin

package constants

import "syscall"

var SysAttr = &syscall.SysProcAttr{
	Setpgid:   true,
	Pdeathsig: syscall.SIGKILL,
}
