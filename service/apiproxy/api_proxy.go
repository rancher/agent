package apiproxy

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"github.com/rancher/agent/utilities/config"
	"github.com/rancher/agent/utilities/utils"
	"net"
	urls "net/url"
	"os"
	"os/exec"
	"strings"
	"github.com/rancher/agent/utilities/constants"
)

func StartUp() error {
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
	toHostIP, err := net.LookupIP(parsed.Host)
	if err != nil {
		return errors.Wrap(err, "Can not look up IPAddress")
	}
	toPort := utils.GetURLPort(url)
	logrus.Infof("Proxying %s:%s -> %s:%s", fromHost, fromPort, toHostIP, toPort)
	listen := fmt.Sprintf("TCP4-LISTEN:%v,fork,bind=%v,reuseaddr", fromPort, fromHost)
	to := fmt.Sprintf("TCP:%v:%v", toHostIP, toPort)
	command := exec.Command("socat", listen, to)
	command.SysProcAttr = constants.SysAttr
	command.Stderr = os.Stderr
	command.Stdout = os.Stdout
	command.Start()
	err = command.Wait()
	return err
}
