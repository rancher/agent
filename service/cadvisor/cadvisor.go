package cadvisor

import (
	"os"
	"os/exec"

	"github.com/rancher/agent/utilities/config"
	"github.com/rancher/agent/utilities/constants"
	"github.com/rancher/agent/utilities/docker"
	"github.com/rancher/agent/utilities/utils"
)

func StartUp() error {
	//TODO: this should be in for loop to keep restarting every 5 seconds
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
	command := exec.Command(args[0], args[1:len(args)]...)
	// need to pdeathsig
	command.SysProcAttr = constants.SysAttr
	command.Stderr = os.Stderr
	command.Stdout = os.Stdout
	command.Start()
	err := command.Wait()
	return err
}

func cadvisorDockerRoot() string {
	return docker.Info.DockerRootDir
}
