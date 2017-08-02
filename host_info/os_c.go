package hostInfo

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/rancher/agent/utils"
)

type OSCollector struct {}

func (o OSCollector) getDockerVersion(verbose bool) map[string]string {
	data := map[string]string{}
	versionData := DockerData.Version
	version := "unknown"
	if verbose && versionData.Version != "" {
		version = fmt.Sprintf("Docker version %v, build %v", versionData.Version, versionData.GitCommit)
	} else if versionData.Version != "" {
		version = utils.SemverTrunk(versionData.Version, 2)
	}
	data["dockerVersion"] = version

	return data
}

func (o OSCollector) GetData() (map[string]interface{}, error) {
	data := map[string]interface{}{}
	osData, err := o.getOS()
	if err != nil {
		return data, errors.Wrap(err, "failed to get OS data")
	}

	for key, value := range o.getDockerVersion( true) {
		data[key] = value
	}
	for key, value := range osData {
		data[key] = value
	}
	return data, nil
}

func (o OSCollector) GetLabels(prefix string) (map[string]string, error) {
	osData, err := o.getOS()
	if err != nil {
		return map[string]string{}, errors.Wrap(err, "failed to get OS data")
	}
	labels := map[string]string{
		fmt.Sprintf("%s.%s", prefix, "docker_version"):       o.getDockerVersion( false)["dockerVersion"],
		fmt.Sprintf("%s.%s", prefix, "linux_kernel_version"): utils.SemverTrunk(osData["kernelVersion"], 2),
	}
	return labels, nil
}

func (o OSCollector) KeyName() string {
	return "osInfo"
}

func (o OSCollector) getOS() (map[string]string, error) {
	data := map[string]string{}
	data["operatingSystem"] = DockerData.Info.OperatingSystem
	kernelVersion, err := getKernelVersion()
	if err != nil {
		return map[string]string{}, errors.Wrap(err, "failed to get kernel version")
	}
	data["kernelVersion"] = kernelVersion

	return data, nil
}
