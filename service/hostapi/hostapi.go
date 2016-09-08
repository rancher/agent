package hostapi

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/rancher/agent/utilities/config"
	"github.com/rancher/agent/utilities/constants"
	"os"
	"os/exec"
	"time"
)

func StartUp() {
	for {
		env := os.Environ()
		env = append(env, fmt.Sprintf("%v=%v", "HOST_API_CATTLE_ACCESS_KEY", config.AccessKey()))
		env = append(env, fmt.Sprintf("%v=%v", "HOST_API_CATTLE_SECRET_KEY", config.SecretKey()))
		url := fmt.Sprintf("http://%v:%v", config.CadvisorIP(), config.CadvisorPort())
		uuid, err := config.DockerUUID()
		if err != nil {
			logrus.Error(err)
			time.Sleep(time.Duration(1) * time.Second)
			continue
		}
		args := []string{
			"-cadvisor-url", url,
			"-logtostderr=true",
			"-ip", config.HostAPIIP(),
			"-port", config.HostAPIPort(),
			"-auth=true",
			"-host-uuid", uuid,
			"-public-key", config.JwtPublicKeyFile(),
			"-cattle-url", config.APIURL(""),
			"-cattle-state-dir", config.ContainerStateDir(),
		}
		logrus.Info(execPath)
		command := exec.Command(execPath, args...)
		command.Env = env
		command.SysProcAttr = constants.SysAttr
		command.Stderr = os.Stderr
		command.Stdout = os.Stdout
		if err := command.Start(); err != nil {
			logrus.Error(err)
		}
		if err := command.Wait(); err != nil {
			logrus.Error(err)
		}
		time.Sleep(time.Duration(5) * time.Second)
	}
}
