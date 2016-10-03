package cadvisor

import (
	"github.com/Sirupsen/logrus"
	"github.com/rancher/agent/utilities/config"
	"github.com/rancher/agent/utilities/constants"
	"github.com/rancher/agent/utilities/docker"
	"github.com/rancher/agent/utilities/utils"
	"golang.org/x/net/context"
	"os"
	"os/exec"
	"time"
)

func StartUp() error {
	for {
		args := []string{
			"cadvisor",
			"-logtostderr=true",
			"-listen_ip", config.CadvisorIP(),
			"-port", config.CadvisorPort(),
			"-housekeeping_interval", config.CadvisorInterval(),
		}
		dockerRoot := cadvisorDockerRoot()
		if len(dockerRoot) > 0 {
			args = append(args, []string{"-docker_root", dockerRoot}...)
		}
		cadvisorOpts := config.CadvisorOpts()
		if len(cadvisorOpts) > 0 {
			args = append(args, utils.SafeSplit(cadvisorOpts)...)
		}
		wrapper := config.CadvisorWrapper()
		if len(wrapper) > 0 {
			args = append([]string{wrapper}, args...)
		} else if _, err := os.Stat("/host/proc/1/ns/mnt"); err == nil {
			args = append([]string{"nsenter", "--mount=/host/proc/1/ns/mnt", "--"}, args...)
		}
		command := exec.Command(args[0], args[1:]...)
		command.SysProcAttr = constants.SysAttr
		if err := command.Start(); err != nil {
			logrus.Error(err)
		}
		if err := command.Wait(); err != nil {
			if err.Error() == "exit status 255" {
				break
			}
		}
		time.Sleep(time.Duration(5) * time.Second)
	}
	return nil
}

func cadvisorDockerRoot() string {
	info, err := docker.GetClient(constants.DefaultVersion).Info(context.Background())
	if err != nil {
		logrus.Error(err)
		return ""
	}
	return info.DockerRootDir
}
