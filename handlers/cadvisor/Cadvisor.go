package cadvisor

import (
	"github.com/Sirupsen/logrus"
	"github.com/rancher/agent/handlers/utils"
	"os"
	"os/exec"
)

func StartUp() error {
	args := []string{
		"cadvisor",
		"-logtostderr=true",
		"-listen_ip", utils.CadvisorIP(),
		"-port", utils.CadvisorPort(),
		"-housekeeping_interval", utils.CadvisorInterval(),
	}
	dockerRoot := utils.CadvisorDockerRoot()
	if len(dockerRoot) > 0 {
		args = append(args, []string{"-docker_root", dockerRoot}...)
	}
	cadvisorOpts := utils.CadvisorOpts()
	if len(cadvisorOpts) > 0 {
		args = append(args, utils.SafeSplit(cadvisorOpts)...)
	}
	wrapper := utils.CadvisorWrapper()
	logrus.Info(wrapper)
	if len(wrapper) > 0 {
		args = append([]string{wrapper}, args...)
	} else if _, err := os.Stat("/host/proc/1/ns/mnt"); err == nil {
		args = append([]string{"nsenter", "--mount=/host/proc/1/ns/mnt", "--"}, args...)
	}
	logrus.Infof("args %v", args)
	command := exec.Command(args[0], args[1:len(args)]...)
	logrus.Infof("check command structure %+v", command)
	command.Stderr = os.Stderr
	command.Stdout = os.Stdout
	command.Start()
	err := command.Wait()
	return err
}
