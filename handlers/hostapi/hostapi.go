package hostapi

import (
	"os"
	"fmt"
	"github.com/rancher/agent/handlers/utils"
	"os/exec"
)

func HostAPIStartUp() error {
	env := os.Environ()
	env = append(env, fmt.Sprintf("%v=%v", "HOST_API_CATTLE_ACCESS_KEY", utils.AccessKey()))
	env = append(env, fmt.Sprintf("%v=%v", "HOST_API_CATTLE_SECRET_KEY", utils.SecretKey()))
	url := fmt.Sprintf("http://%v:%v", utils.CadvisorIP(), utils.CadvisorPort())
	args := []string{
		"-cadvisor-url", url,
		"-logtostderr=true",
		"-ip", utils.HostAPIIP(),
		"-port", utils.HostAPIPort(),
		"-auth=true",
		"-host-uuid", utils.DockerUUID(),
		"-public-key", utils.JwtPublicKeyFile(),
		"-cattle-url", utils.ApiURL(""),
		"-cattle-state-dir", utils.ContainerStateDir(),
	}
	command := exec.Command("host-api", args...)
	command.Env = env
	command.Start()
	err := command.Wait()
	return err
}