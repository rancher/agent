package hostInfo

import (
	"fmt"
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"golang.org/x/net/context"
	"runtime"
)

type OSCollector struct {
	client *client.Client
}

func (o OSCollector) dockerVersionRequest() types.Version {
	if o.client != nil {
		version, _ := o.client.ServerVersion(context.Background())
		return version
	}
	return types.Version{}
}

func (o OSCollector) getDockerVersion(verbose bool) map[string]interface{} {
	data := map[string]interface{}{}
	if runtime.GOOS == "linux" {
		verResp := o.dockerVersionRequest()
		version := "unknown"
		if verbose && verResp.Version != "" {
			version = fmt.Sprintf("Docker version %v, build %v", verResp.Version, verResp.GitCommit)
		} else if verResp.Version != "" {
			version = semverTrunk(verResp.Version, 0)
		}
		data["dockerVersion"] = version
	}
	return data
}

func (o OSCollector) getOS() map[string]interface{} {
	data := map[string]interface{}{}
	if runtime.GOOS == "linux" {
		if o.client != nil {
			info, _ := o.client.Info(context.Background())
			data["operatingSystem"] = info.OperatingSystem
			data["kernelVersion"] = getKernelVersion()
		}
	}
	return data
}

func (o OSCollector) GetData() map[string]interface{} {
	data := map[string]interface{}{}
	for key, value := range o.getOS() {
		data[key] = value
	}
	return data
}

func (o OSCollector) GetLabels(prefix string) map[string]string {
	if runtime.GOOS == "linux" {
		labels := map[string]string{
			fmt.Sprintf("%s.%s", prefix, "docker_version"):       o.getDockerVersion(false)["dockerVersion"].(string),
			fmt.Sprintf("%s.%s", prefix, "linux_kernel_version"): semverTrunk(o.getOS()["kernelVersion"].(string), 2),
		}
		return labels
	}
	return map[string]string{}
}

func (o OSCollector) KeyName() string {
	return "osInfo"
}
