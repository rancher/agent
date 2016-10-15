//+build windows

package utils

import (
	"bufio"
	"github.com/Sirupsen/logrus"
	"github.com/docker/engine-api/types"
	"github.com/rancher/agent/utilities/constants"
	"github.com/rancher/agent/utilities/docker"
	"golang.org/x/net/context"
	"regexp"
	"strings"
	"time"
)

func getIP(inspect types.ContainerJSON) string {
	containerID := inspect.ID
	client := docker.GetClient(constants.DefaultVersion)
	execConfig := types.ExecConfig{
		AttachStdout: true,
		AttachStdin:  true,
		AttachStderr: true,
		Privileged:   true,
		Tty:          false,
		Detach:       false,
		Cmd:          []string{"powershell", "ipconfig"},
	}
	ip := ""
	// waiting for the DHCP to assign IP address. Testing purpose. May try multiple times until ip address arrives
	time.Sleep(time.Duration(2) * time.Second)
	execObj, err := client.ContainerExecCreate(context.Background(), containerID, execConfig)
	if err != nil {
		logrus.Error(err)
		return ""
	}
	hijack, err := client.ContainerExecAttach(context.Background(), execObj.ID, execConfig)
	if err != nil {
		logrus.Error(err)
		return ""
	}
	scanner := bufio.NewScanner(hijack.Reader)
	for scanner.Scan() {
		output := scanner.Text()
		if strings.Contains(output, "IPv4 Address") {
			ip = regexp.MustCompile("(?:[0-9]{1,3}\\.){3}[0-9]{1,3}$").FindString(output)
		}
	}
	hijack.Close()
	return ip
}
