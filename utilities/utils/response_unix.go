//+build !windows

package utils

import (
	"github.com/docker/docker/api/types"
	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	"github.com/rancher/agent/utilities/constants"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
	"time"
)

const (
	cniLabels = "io.rancher.network.cni"
	linkName  = "eth0"
)

func getIP(inspect types.ContainerJSON, c *cache.Cache) (string, error) {
	if val, ok := inspect.Config.Labels[cniLabels]; ok && val == "true" && c != nil {
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
	endTime := time.Now().Add(time.Duration(1) * time.Minute)
	initTime := time.Duration(250) * time.Millisecond
	maxTime := time.Duration(2) * time.Second
	for {
		// back off
		nsHandler, err := netns.GetFromPid(inspect.State.Pid)
		if err != nil {
			continue
		}
		handler, err := netlink.NewHandleAt(nsHandler)
		if err != nil {
			continue
		}
		link, err := handler.LinkByName(linkName)
		if err != nil {
			continue
		}
		addrs, err := handler.AddrList(link, netlink.FAMILY_V4)
		if err != nil {
			continue
		}
		if len(addrs) > 0 {
			return addrs[0].IP.String(), nil
		}
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
