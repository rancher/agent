package hostInfo

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/rancher/agent/model"
	"github.com/rancher/agent/utilities/constants"
	"github.com/rancher/agent/utilities/utils"
)

type OSCollector struct {
	InfoData model.InfoData
}

func (o OSCollector) getDockerVersion(infoData model.InfoData, verbose bool) map[string]string {
	data := map[string]string{}
	versionData := infoData.Version
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
	infoData := o.InfoData
	data := map[string]interface{}{}
	osData, err := o.getOS(infoData)
	if err != nil {
		return data, errors.Wrap(err, constants.OSGetDataError+"failed to get OS data")
	}

	for key, value := range o.getDockerVersion(infoData, true) {
		data[key] = value
	}
	for key, value := range osData {
		data[key] = value
	}
	return data, nil
}

func (o OSCollector) GetLabels(prefix string) (map[string]string, error) {
	osData, err := o.getOS(o.InfoData)
	if err != nil {
		return map[string]string{}, errors.Wrap(err, constants.OSGetDataError+"failed to get OS data")
	}
	labels := map[string]string{
		fmt.Sprintf("%s.%s", prefix, "docker_version"):                 o.getDockerVersion(o.InfoData, false)["dockerVersion"],
		fmt.Sprintf("%s.%s%s", prefix, getOSName(), "_kernel_version"): utils.SemverTrunk(osData["kernelVersion"], 2),
		fmt.Sprintf("%s.%s", prefix, "os"):                             getOSName(),
	}
	if getOSName() == "windows" {
		labels["io.rancher.infra_service.healthcheck.deploy"] = "never"
	}
	return labels, nil
}

func (o OSCollector) KeyName() string {
	return "osInfo"
}

func (o OSCollector) getOS(infoData model.InfoData) (map[string]string, error) {
	data := map[string]string{}
	data["operatingSystem"] = infoData.Info.OperatingSystem
	kernelVersion, err := getKernelVersion()
	if err != nil {
		return map[string]string{}, errors.Wrap(err, constants.GetOSError+"failed to get kernel version")
	}
	data["kernelVersion"] = kernelVersion

	return data, nil
}
