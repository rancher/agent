package hostInfo

func (o OSDataGetter) GetOS(infoData model.InfoData) (map[string]string, error) {
	data := map[string]interface{}{}
	info := infoData
	data["operatingSystem"] = info.OperatingSystem
	kv, err := utils.GetWindowsKernelVersion()
	if err != nil {
		return map[string]string{}, errors.Wrap(err, constants.GetWindowsOSError)
	}
	data["kernelVersion"] = kv
	return data, nil
}
