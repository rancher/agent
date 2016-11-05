// +build linux freebsd solaris openbsd darwin

package compute

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

// This approach was adopted from Weave's weaveproxy component. Proper credit goes to them.

var (
	waitProg         = "/.r/r"
	waitVolume       = "rancher-cni"
	waitVolumeTarget = "/.r"
)

func modifyForCNI(c *client.Client, container *container.Config, hostConfig *container.HostConfig) error {
	if container.Labels[cniWaitLabel] != "true" {
		return nil
	}
	if err := setWaitEntrypoint(c, container); err != nil {
		return err
	}
	return setWaitVolume(hostConfig, waitVolume, waitVolumeTarget, "ro")
}

func setWaitVolume(hostConfig *container.HostConfig, source, target, mode string) error {
	var binds []string
	for _, bind := range hostConfig.Binds {
		s := strings.Split(bind, ":")
		if len(s) >= 2 && s[1] == target {
			continue
		}
		binds = append(binds, bind)
	}
	hostConfig.Binds = append(hostConfig.Binds, fmt.Sprintf("%s:%s:%s", source, target, mode))
	return nil
}

func setWaitEntrypoint(c *client.Client, container *container.Config) error {
	if len(container.Entrypoint) == 0 {
		image, _, err := c.ImageInspectWithRaw(context.Background(), container.Image)
		if err != nil {
			return err
		}

		if len(container.Cmd) == 0 {
			container.Cmd = image.Config.Cmd
		}

		if container.Entrypoint == nil {
			container.Entrypoint = image.Config.Entrypoint
		}
	}

	if len(container.Entrypoint) == 0 && len(container.Cmd) == 0 {
		// No command, just let docker complain about it on start
		return nil
	}

	if len(container.Entrypoint) == 0 || container.Entrypoint[0] != waitProg {
		container.Entrypoint = append([]string{waitProg}, container.Entrypoint...)
	}

	return nil
}
