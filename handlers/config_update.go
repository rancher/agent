package handlers

import (
	"fmt"
	"github.com/rancher/agent/core/progress"
	"github.com/rancher/agent/utilities/config"
	"github.com/rancher/agent/utilities/utils"
	revents "github.com/rancher/event-subscriber/events"
	"github.com/rancher/go-rancher/v2"
	"net/http"
	"os"
	"os/exec"
	"runtime"
)

type ConfigUpdateHandler struct {
}

func (h *ConfigUpdateHandler) ConfigUpdate(event *revents.Event, cli *client.RancherClient) error {
	if runtime.GOOS == "windows" {
		// if os is windows, do a fake reply and manually touch some urls as @ibuildthecloud suggested
		items := getItemList(event.Data)
		urls := []string{}
		for _, item := range items {
			url := config.APIURL("") + fmt.Sprintf("/configcontent/%v?version=latest", item)
			urls = append(urls, url)
		}
		for _, url := range urls {
			req, err := http.NewRequest("PUT", url, nil)
			if err != nil {
				return err
			}
			req.SetBasicAuth(config.AccessKey(), config.SecretKey())
			httpClient := &http.Client{}
			_, err = httpClient.Do(req)
			if err != nil {
				return err
			}
		}
		return reply(map[string]interface{}{}, event, cli)
	}
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

	retcode := 0

	command := exec.Command(config.Sh(), args...)
	command.Env = env
	command.Dir = home
	output, err := command.CombinedOutput()
	if err != nil {
		retcode = utils.GetExitCode(err)
	}
	if retcode != 0 {
		pro := &progress.Progress{Request: event, Client: cli}
		pro.Update("config update failed", "no", map[string]interface{}{
			"exitCode": retcode,
			"output":   string(output),
		})
		return nil
	}
	return reply(map[string]interface{}{}, event, cli)
}

func getItemList(data map[string]interface{}) []string {
	items := []string{}
	maps, ok := data["items"].([]interface{})
	if !ok {
		return items
	}
	for _, m := range maps {
		m1, ok := m.(map[string]interface{})
		if !ok {
			continue
		}
		name, ok := m1["name"].(string)
		if !ok {
			continue
		}
		items = append(items, name)
	}
	return items
}
