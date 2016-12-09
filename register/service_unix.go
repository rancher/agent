//+build !windows

package register

func Init(register string, unregister bool) error { return nil }

func NotifyShutdown(err error) {}
