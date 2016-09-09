package hostInfo

import (
	"github.com/pkg/errors"
	"github.com/rancher/agent/model"
	"github.com/rancher/agent/utilities/constants"
	"github.com/rancher/agent/utilities/utils"
)

func (o OSDataGetter) GetOS(infoData model.InfoData) (map[string]string, error) {
	data := map[string]string{}
	info := infoData.Info
	data["operatingSystem"] = info.OperatingSystem
	kv, err := utils.GetWindowsKernelVersion()
	if err != nil {
		return map[string]string{}, errors.Wrap(err, constants.GetWindowsOSError)
	}
	data["kernelVersion"] = kv
	return data, nil
}
