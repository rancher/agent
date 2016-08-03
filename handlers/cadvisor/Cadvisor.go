package cadvisor

import (
	"github.com/rancher/agent/handlers/utils"
	"os"
	"os/exec"
)

func CadvisorStartUp() error {
	args := []string{"cadvisor", "-logtostderr=true", "-listen_ip", utils.CadvisorIP(), "-port", utils.CadvisorPort(), "-housekeeping_interval",
		utils.CadvisorInterval()}
	dockerRoot := utils.CadvisorDockerRoot()
	if len(dockerRoot) > 0 {
		args = append([]string{"-docker_root"}, args...)
	}
	cadvisorOpts := utils.CadvisorOpts()
	if len(cadvisorOpts) > 0 {
		args = append(args, utils.SafeSplit(cadvisorOpts)...)
	}
	wrapper := utils.CadvisorWrapper()
	if len(wrapper) > 0 {
		args = append([]string{wrapper}, args...)
	} else if _, err := os.Stat("/host/proc/1/ns/mnt"); err == nil {
		args = append([]string{"nsenter", "--mount=/host/proc/1/ns/mnt", "--"}, args)
	}
	command := exec.Command(args[0], args[1:]...)
	command.Start()
	err := command.Wait()
	return err
}
