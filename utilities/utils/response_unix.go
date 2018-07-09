//+build !windows

package utils

import (
	"encoding/json"
	"io/ioutil"
	"net"
	"os"
	"path"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	"github.com/rancher/agent/utilities/constants"
	"github.com/rancher/log"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

const (
	cniLabels       = "io.rancher.cni.network"
	linkName        = "eth0"
	cniStateBaseDir = "/var/lib/rancher/state/cni"
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
	// if container is stopped just return empty ip
	if inspect.State.Pid == 0 {
		return "", nil
	}
	endTime := time.Now().Add(30 * time.Second)
	initTime := 250 * time.Millisecond
	maxTime := 2 * time.Second
	for {
		if ip, cniError := getIPFromStateFile(inspect); cniError != nil {
			return "", cniError
		} else if ip != "" {
			return ip, nil
		}

		ip, err := getIPForPID(inspect.State.Pid)
		if err != nil || ip != "" {
			return ip, err
		}

		log.Debugf("Sleeping %v (%v remaining) waiting for IP on %s", initTime, endTime.Sub(time.Now()), inspect.ID)
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

func getIPFromStateFile(inspect types.ContainerJSON) (string, error) {
	if inspect.ID == "" || inspect.State == nil || inspect.State.StartedAt == "" {
		return "", nil
	}
	filename := path.Join(cniStateBaseDir, inspect.ID, inspect.State.StartedAt)

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Warnf("Error reading cni state file %v: %v. Falling back to container inspection logic.", filename, err)
		}
		return "", nil
	}

	var state cniState
	if err := json.Unmarshal(data, &state); err != nil {
		log.Warnf("Error unmarshalling cni state data %s: %v. Falling back to container inspection logic.", data, err)
		return "", nil
	}

	if state.Error != "" {
		return "", errors.New(state.Error)
	}

	if state.IP4.IP != "" {
		ip, _, err := net.ParseCIDR(state.IP4.IP)
		if err != nil {
			return "", errors.Wrapf(err, "Unable to parse recorded IP address %v", state.IP4.IP)
		}
		return ip.String(), nil
	}
	return "", nil
}

type cniState struct {
	Error string
	IP4   struct {
		IP string
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
