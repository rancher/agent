package hostInfo

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/rancher/agent/utilities/utils"
)

type OSCollector struct {
	dataGetter OSInfoGetter
	GOOS       string
}

type OSInfoGetter interface {
	GetOS() map[string]interface{}
	GetDockerVersion(bool) map[string]interface{}
}

type OSDataGetter struct{}

func (o OSDataGetter) GetDockerVersion(verbose bool) map[string]interface{} {
	data := map[string]interface{}{}
	verResp, err := utils.DockerVersionRequest()
	if err != nil {
		logrus.Error(err)
	} else {
		version := "unknown"
		if verbose && verResp.Version != "" {
			version = fmt.Sprintf("Docker version %v, build %v", verResp.Version, verResp.GitCommit)
		} else if verResp.Version != "" {
			version = utils.SemverTrunk(verResp.Version, 2)
		}
		data["dockerVersion"] = version
	}
	return data
}

func (o OSDataGetter) GetOS() map[string]interface{} {
	data := map[string]interface{}{}
	info, err := utils.GetInfo()
	if err != nil {
		logrus.Error(err)
	} else {
		data["operatingSystem"] = info.OperatingSystem
		data["kernelVersion"] = utils.GetKernelVersion()
	}
	return data
}

func (o OSCollector) GetData() map[string]interface{} {
	data := map[string]interface{}{}
	if o.GOOS == "linux" {
		for key, value := range o.dataGetter.GetOS() {
			data[key] = value
		}
		for key, value := range o.dataGetter.GetDockerVersion(true) {
			data[key] = value
		}
	}
	return data
}

func (o OSCollector) GetLabels(prefix string) map[string]string {
	labels := map[string]string{
		fmt.Sprintf("%s.%s", prefix, "docker_version"):       utils.InterfaceToString(o.dataGetter.GetDockerVersion(false)["dockerVersion"]),
		fmt.Sprintf("%s.%s", prefix, "linux_kernel_version"): utils.SemverTrunk(utils.InterfaceToString(o.dataGetter.GetOS()["kernelVersion"]), 2),
	}
	return labels
}

func (o OSCollector) KeyName() string {
	return "osInfo"
}
