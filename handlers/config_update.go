package handlers

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/rancher/agent/handlers/progress"
	"github.com/rancher/agent/handlers/utils"
	revents "github.com/rancher/go-machine-service/events"
	"github.com/rancher/go-rancher/client"
	"os"
	"os/exec"
)

func ConfigUpdate(event *revents.Event, cli *client.RancherClient) error {
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
		if name != "pyagent" || utils.ConfigUpdatePyagent() {
			itemNames = append(itemNames, name)
		}
	}
	home := utils.Home()
	env := os.Environ()
	env = append(env, fmt.Sprintf("%v=%v", "CATTLE_ACCESS_KEY", utils.AccessKey()))
	env = append(env, fmt.Sprintf("%v=%v", "CATTLE_SECRET_KEY", utils.SecretKey()))
	env = append(env, fmt.Sprintf("%v=%v", "CATTLE_HOME", home))
	args := itemNames

	retcode := -1

	command := exec.Command(utils.ConfigSh(), args...)
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
