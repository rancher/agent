package handlers

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/rancher/agent/core/progress"
	"github.com/rancher/agent/utilities/config"
	"github.com/rancher/agent/utilities/utils"
	revents "github.com/rancher/event-subscriber/events"
	"github.com/rancher/go-rancher/v2"
	"os"
	"os/exec"
)

type ConfigUpdateHandler struct {
}

func (h *ConfigUpdateHandler) ConfigUpdate(event *revents.Event, cli *client.RancherClient) error {
	if event.Name != "config.update" || event.ReplyTo == "" {
		return nil
	}

	if len(utils.InterfaceToArray(event.Data["items"])) == 0 {
		return reply(map[string]interface{}{}, event, cli)
	}
	itemNames := []string{}

	for _, v := range utils.InterfaceToArray(event.Data["items"]) {
		item := utils.InterfaceToMap(v)
		name := utils.InterfaceToString(item["name"])
		if name != "pyagent" || config.UpdatePyagent() {
			itemNames = append(itemNames, name)
		}
	}
	home := config.Home()
	env := os.Environ()
	env = append(env, fmt.Sprintf("%v=%v", "CATTLE_ACCESS_KEY", config.AccessKey()))
	env = append(env, fmt.Sprintf("%v=%v", "CATTLE_SECRET_KEY", config.SecretKey()))
	env = append(env, fmt.Sprintf("%v=%v", "CATTLE_HOME", home))
	args := itemNames

	retcode := -1

	command := exec.Command(config.Sh(), args...)
	command.Env = env
	command.Dir = home
	output, err := command.Output()
	if err != nil {
		logrus.Error(err)
	} else {
		retcode = 0
	}
	if retcode == 0 {
		return reply(map[string]interface{}{
			"exitCode": retcode,
			"output":   string(output),
		}, event, cli)
	}
	pro := &progress.Progress{Request: event, Client: cli}
	pro.Update("config update failed")
	return nil
}
