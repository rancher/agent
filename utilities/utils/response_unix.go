//+build !windows

package utils

import (
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/api/types"
	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	"github.com/rancher/agent/utilities/constants"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

const (
	cniLabels = "io.rancher.cni.network"
	StartOnce = "io.rancher.container.start_once"
	linkName  = "eth0"
)

func getIP(inspect types.ContainerJSON, c *cache.Cache) (string, error) {
	if inspect.Config.Labels[cniLabels] != "" && c != nil {
		cacheIP, ok := c.Get(inspect.Config.Labels[constants.UUIDLabel])
		if ok && InterfaceToString(cacheIP) == "error" {
			c.Delete(inspect.Config.Labels[constants.UUIDLabel])
			return "", errors.New("Timeout getting IP address")
		} else if ok {
			c.Delete(inspect.Config.Labels[constants.UUIDLabel])
			return InterfaceToString(cacheIP), nil
		}
		ip, err := lookUpIP(inspect)
		if err != nil {
			c.Add(inspect.Config.Labels[constants.UUIDLabel], "error", cache.DefaultExpiration)
			return "", err
		}
		c.Add(inspect.Config.Labels[constants.UUIDLabel], ip, cache.DefaultExpiration)
		return ip, nil
	}
	return inspect.NetworkSettings.IPAddress, nil
}

func lookUpIP(inspect types.ContainerJSON) (string, error) {
	endTime := time.Now().Add(30 * time.Second)
	initTime := 250 * time.Millisecond
	maxTime := 2 * time.Second
	for {
		ip, err := getIPForPID(inspect.State.Pid)
		if err != nil || ip != "" {
			// if it has a error and it is start-once container, ignore the error
			if inspect.Config.Labels[StartOnce] == "true" {
				return ip, nil
			}
			return ip, err
		}

		logrus.Debugf("Sleeping %v (%v remaining) waiting for IP on %s", initTime, endTime.Sub(time.Now()), inspect.ID)
		time.Sleep(initTime)
		initTime = initTime * 2
		if initTime.Seconds() > maxTime.Seconds() {
			initTime = maxTime
		}
		if time.Now().After(endTime) {
			return "", errors.New("Timeout getting IP address")
		}
	}
}

func getIPForPID(pid int) (string, error) {
	nsHandler, err := netns.GetFromPid(pid)
	if err != nil {
		return "", err
	}
	defer nsHandler.Close()
	handler, err := netlink.NewHandleAt(nsHandler)
	if err != nil {
		return "", err
	}
	defer handler.Delete()
	link, err := handler.LinkByName(linkName)
	if err != nil {
		// Don't return error, it's expected this may fail until iface is created
		return "", nil
	}
	addrs, err := handler.AddrList(link, netlink.FAMILY_V4)
	if err != nil {
		return "", err
	}
	if len(addrs) > 0 {
		return addrs[0].IP.String(), nil
	}
	return "", nil
}
