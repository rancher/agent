package utils

import (
	revents "github.com/rancher/go-machine-service/events"
	/*
		"path"
		"strings"
		"strconv"
		"github.com/rancher/agent/handlers/marshaller"
		"os"
		"fmt"
		"github.com/Sirupsen/logrus"
		"os/exec"
		"time"
		"github.com/mitchellh/mapstructure"
	*/)

func NsExec(pid int, event *revents.Event) (int, string, *revents.Event) {
	/*
		script := path.Join(home(), "events", strings.Split(event.Name, ";")[0])
		cmd := []string{"nsenter", "-F", "-m", "-u", "-i", "-n", "-p", "-t", strconv.Itoa(pid), "--", script}
		input, _ := marshaller.ToString(event)
		var data revents.Event

		envmap := map[string]string{}
		file, fileErr := os.Open(fmt.Sprintf("/proc/%v/environ", pid))
		if fileErr != nil {
			logrus.Error(fileErr)
		}
		for _, line := range strings.Split(ReadBuffer(file), "\n") {
			if len(line) == 0 {
				continue
			}
			kv := strings.SplitAfterN(line, "=", 1)
			if strings.HasPrefix(kv[0], "CATTLE") {
				envmap[kv[0]] = kv[1]
			}
		}
		envmap["PATH"] = os.Getenv("PATH")
		envmap["CATTLE_CONFIG_URL"] = configURL()
		envs := []string{}
		for key, value := range envmap {
			envs = append(envs, fmt.Sprintf("%s=%s", key, value))
		}
		retcode := -1
		output := ""
		// go doc shows that the following function may not run in window
		for i := 0; i < 3 ; i++ {
			{
				command := exec.Command("delegate", cmd)
				stdin, _ := command.StdinPipe()
				stdin.Write(input)
				command.Env = envs
				command.Start()
				stdout, _ := command.StdoutPipe()
				output = ReadBuffer(stdout)
				err := command.Wait()
				if err == nil {
					retcode = 0
					break
				}
				stdin.Close()
				stdout.Close()
			}
			{
				exCMD := append(cmd[:len(cmd)-1], "/usr/bin/test", "-e", script)
				existCMD := exec.Command("delegate", exCMD)
				existCMD.Env = envs
				stdin, _ := existCMD.StdinPipe()
				stdin.Write(input)
				existCMD.Start()
				stdout, _ := existCMD.StdoutPipe()
				output = ReadBuffer(stdout)
				err := existCMD.Wait()
				if err == nil {
					retcode = 0
					break
				}
				stdin.Close()
				stdout.Close()
			}
			time.Sleep(time.Second)
		}
		if retcode == 0 {
			return retcode, output, nil
		}
		text := []string{}
		for _, line := range strings.Split(output, "\n") {
			if strings.HasPrefix(line, "{") {
				buffer := marshaller.FromString(line)
				mapstructure.Decode(buffer, &data)
				break
			}
			text = append(text, line)
		}
		return retcode, strings.Join(text, ""), &data
	*/
	return 0, "", nil
}
