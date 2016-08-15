package hostapi

import (
	"fmt"
	"github.com/rancher/agent/utilities/config"
	"os"
	"os/exec"
	"syscall"
)

func StartUp() error {
	env := os.Environ()
	env = append(env, fmt.Sprintf("%v=%v", "HOST_API_CATTLE_ACCESS_KEY", config.AccessKey()))
	env = append(env, fmt.Sprintf("%v=%v", "HOST_API_CATTLE_SECRET_KEY", config.SecretKey()))
	url := fmt.Sprintf("http://%v:%v", config.CadvisorIP(), config.CadvisorPort())
	args := []string{
		"-cadvisor-url", url,
		"-logtostderr=true",
		"-ip", config.HostAPIIP(),
		"-port", config.HostAPIPort(),
		"-auth=true",
		"-host-uuid", config.DockerUUID(),
		"-public-key", config.JwtPublicKeyFile(),
		"-cattle-url", config.APIURL(""),
		"-cattle-state-dir", config.ContainerStateDir(),
	}
	command := exec.Command("host-api", args...)
	command.Env = env
	command.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
	command.Stderr = os.Stderr
	command.Stdout = os.Stdout
	command.Start()
	err := command.Wait()
	return err
}
