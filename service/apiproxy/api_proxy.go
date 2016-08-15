package apiproxy

import (
	"strings"
	"github.com/rancher/agent/utilities/utils"
	"github.com/rancher/agent/utilities/config"
	urls "net/url"
	"net"
	"github.com/Sirupsen/logrus"
	"fmt"
	"os/exec"
	"syscall"
	"os"
	"github.com/pkg/errors"
)

func StartUp() (error) {
	url := config.URL()

	if !strings.Contains(url, "localhost") {
		return nil
	}

	parsed, err := urls.Parse(url)

	if err != nil {
		return nil
	}

	fromHost := config.APIProxyListenHost()
	fromPort := config.APIProxyListenPort()
	toHostIp, err := net.LookupIP(parsed.Host)
	if err != nil {
		return errors.Wrap(err, "Can not look up IPAddress")
	}
	toPort := utils.GetURLPort(url)
	logrus.Infof("Proxying %s:%s -> %s:%s", fromHost, fromPort, toHostIp, toPort)
	listen := fmt.Sprintf("TCP4-LISTEN:%v,fork,bind=%v,reuseaddr", fromPort, fromHost)
	to := fmt.Sprintf("TCP:%v:%v", toHostIp, toPort)
	command := exec.Command("socat", listen, to)
	command.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
	command.Stderr = os.Stderr
	command.Stdout = os.Stdout
	command.Start()
	err = command.Wait()
	return err
}
