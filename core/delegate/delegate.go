package delegate

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/rancher/agent/core/marshaller"
	"github.com/rancher/agent/utilities/config"
	"github.com/rancher/agent/utilities/constants"
	"github.com/rancher/agent/utilities/utils"
	revents "github.com/rancher/event-subscriber/events"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"
)

func NsExec(pid int, event *revents.Event) (int, string, map[string]interface{}, error) {
	script := path.Join(config.Home(), "events", strings.Split(event.Name, ";")[0])
	cmd := []string{"-F", "-m", "-u", "-i", "-n", "-p", "-t", strconv.Itoa(pid), "--", script}
	input, err := marshaller.ToString(event)
	if err != nil {
		return 1, "", map[string]interface{}{}, errors.Wrap(err, constants.NsExecError+"failed to marshall data")
	}
	data := map[string]interface{}{}

	envmap := map[string]string{}
	file, fileErr := os.Open(fmt.Sprintf("/proc/%v/environ", pid))
	if fileErr != nil {
		return 1, "", map[string]interface{}{}, errors.Wrap(err, constants.NsExecError+"failed to open environ files")
	}
	for _, line := range strings.Split(utils.ReadBuffer(file), "\x00") {
		if len(line) == 0 {
			continue
		}
		kv := strings.SplitN(line, "=", 2)
		if len(kv) == 2 && strings.HasPrefix(kv[0], "CATTLE") {
			envmap[kv[0]] = kv[1]
		}
	}
	envmap["PATH"] = os.Getenv("PATH")
	envmap["CATTLE_CONFIG_URL"] = config.URL()
	envs := []string{}
	for key, value := range envmap {
		envs = append(envs, fmt.Sprintf("%s=%s", key, value))
	}
	retcode := -1
	output := []byte{}
	for i := 0; i < 3; i++ {
		command := exec.Command("nsenter", cmd...)
		command.Env = envs
		command.Stdin = strings.NewReader(string(input))
		buffer, err := command.CombinedOutput()
		if err == nil {
			retcode, output = 0, buffer
			break
		}

		exCMD := append(cmd[:len(cmd)-1], "/usr/bin/test", "-e", script)
		existCMD := exec.Command("nsenter", exCMD...)
		existCMD.Env = envs
		_, err1 := existCMD.CombinedOutput()
		existCMD.Wait()
		if err1 == nil {
			output = buffer
			break
		}

		time.Sleep(time.Duration(1) * time.Second)
	}
	if retcode != 0 {
		return retcode, string(output), map[string]interface{}{}, err
	}
	text := []string{}
	for _, line := range strings.Split(string(output), "\n") {
		if strings.HasPrefix(line, "{") {
			buffer := marshaller.FromString(line)
			data = buffer
			break
		}
		text = append(text, line)
	}
	return retcode, strings.Join(text, ""), data, nil
}
