// +build linux freebsd solaris openbsd darwin

package hostInfo

import (
	"github.com/pkg/errors"
	"github.com/rancher/agent/model"
	"github.com/rancher/agent/utilities/constants"
	"github.com/rancher/agent/utilities/utils"
)

func (o OSDataGetter) GetOS(infoData model.InfoData) (map[string]string, error) {
	data := map[string]string{}
	data["operatingSystem"] = infoData.Info.OperatingSystem
	kernelVersion, err := utils.GetKernelVersion()
	if err != nil {
		return map[string]string{}, errors.Wrap(err, constants.GetOSError+"failed to get kernel version")
	}
	data["kernelVersion"] = kernelVersion

	return data, nil
}
