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
	GetWindowsOS() map[string]interface{}
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
	info := utils.GetInfo()
	data["operatingSystem"] = info.OperatingSystem
	data["kernelVersion"] = utils.GetKernelVersion()

	return data
}

func (o OSDataGetter) GetWindowsOS() map[string]interface{} {
	data := map[string]interface{}{}
	info := utils.GetInfo()
	data["operatingSystem"] = info.OperatingSystem
	kv, err := utils.GetWindowsKernelVersion()
	if err == nil {
		data["kernelVersion"] = kv
	} else {
		logrus.Error(err)
	}
	return data
}

func (o OSCollector) GetData() map[string]interface{} {
	data := map[string]interface{}{}
	for key, value := range o.dataGetter.GetDockerVersion(true) {
		data[key] = value
	}
	if o.GOOS == "linux" {
		for key, value := range o.dataGetter.GetOS() {
			data[key] = value
		}
	} else if o.GOOS == "windows" {
		for key, value := range o.dataGetter.GetWindowsOS() {
			data[key] = value
		}
	}
	return data
}

func (o OSCollector) GetLabels(prefix string) map[string]string {
	labels := map[string]string{}
	if o.GOOS == "linux" {
		labels = map[string]string{
			fmt.Sprintf("%s.%s", prefix, "docker_version"):       utils.InterfaceToString(o.dataGetter.GetDockerVersion(false)["dockerVersion"]),
			fmt.Sprintf("%s.%s", prefix, "linux_kernel_version"): utils.SemverTrunk(utils.InterfaceToString(o.dataGetter.GetOS()["kernelVersion"]), 2),
		}
	} else if o.GOOS == "windows" {
		labels = map[string]string{
			fmt.Sprintf("%s.%s", prefix, "docker_version"):         utils.InterfaceToString(o.dataGetter.GetDockerVersion(false)["dockerVersion"]),
			fmt.Sprintf("%s.%s", prefix, "windows_kernel_version"): utils.InterfaceToString(o.dataGetter.GetWindowsOS()["kernelVersion"]),
		}
	}

	return labels
}

func (o OSCollector) KeyName() string {
	return "osInfo"
}
