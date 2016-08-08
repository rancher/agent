package apiproxy

import (
	"github.com/rancher/agent/handlers/utils"
	"strings"
)

func StartUp() {
	url := utils.ConfigURL()

	if !strings.Contains(url, "localhost") {
		return
	}
	/*
		parsed, _ := urls.Parse(url)

		fromHost := utils.ApiProxyListenHost()
		fromPort := utils.ApiProxyListenPort()
		toHostIp :=
	*/
}
