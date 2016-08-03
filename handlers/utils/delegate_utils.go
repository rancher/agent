package utils

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/rancher/agent/handlers/marshaller"
	revents "github.com/rancher/go-machine-service/events"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"
)

func NsExec(pid int, event *revents.Event) (int, string, map[string]interface{}) {
	script := path.Join(Home(), "events", strings.Split(event.Name, ";")[0])
	cmd := []string{"-F", "-m", "-u", "-i", "-n", "-p", "-t", strconv.Itoa(pid), "--", script}
	input, _ := marshaller.ToString(event)
	data := map[string]interface{}{}

	envmap := map[string]string{}
	file, fileErr := os.Open(fmt.Sprintf("/proc/%v/environ", pid))
	if fileErr != nil {
		logrus.Error(fileErr)
	}
	for _, line := range strings.Split(ReadBuffer(file), "\x00") {
		if len(line) == 0 {
			continue
		}
		kv := strings.SplitN(line, "=", 2)
		if len(kv) == 2 && strings.HasPrefix(kv[0], "CATTLE") {
			envmap[kv[0]] = kv[1]
		}
	}
	envmap["PATH"] = os.Getenv("PATH")
	envmap["CATTLE_CONFIG_URL"] = ConfigURL()
	envs := []string{}
	for key, value := range envmap {
		envs = append(envs, fmt.Sprintf("%s=%s", key, value))
	}
	retcode := -1
	output := []byte{}
	// go doc shows that the following function may not run in window
	for i := 0; i < 3; i++ {
		command := exec.Command("nsenter", cmd...)
		command.Env = envs
		logrus.Infof("input string %v", string(input))
		command.Stdin = strings.NewReader(string(input))
		buffer, err := command.Output()
		if err != nil {
			logrus.Error(err)
		} else {
			retcode, output = 0, buffer
			break
		}

		exCMD := append(cmd[:len(cmd)-1], "/usr/bin/test", "-e", script)
		existCMD := exec.Command("nsenter", exCMD...)
		existCMD.Env = envs
		_, err1 := existCMD.Output()
		existCMD.Wait()
		if err1 == nil {
			break
		} else {
			logrus.Error(err1)
		}

		time.Sleep(time.Duration(1) * time.Second)

	}
	if retcode != 0 {
		return retcode, string(output), map[string]interface{}{}
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
	return retcode, strings.Join(text, ""), data
}
