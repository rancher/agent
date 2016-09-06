package hostInfo

func (c CPUCollector) getCPUInfo() map[string]interface{} {
	data := map[string]interface{}{}
	command := exec.Command("PowerShell", "wmic", "cpu", "get", "Name")
	output, err := command.Output()
	if err == nil {
		ret := strings.Split(string(output), "\n")[1]
		data["modelName"] = ret
		pattern := "([0-9\\.]+)\\s?GHz"
		freq := regexp.MustCompile(pattern).FindString(ret)
		if freq != "" {
			ghz := strings.TrimSpace(freq[:len(freq)-3])
			if ghz != "" {
				mhz, _ := strconv.ParseFloat(ghz, 64)
				data["mhz"] = mhz * 1000
			}
		}
	} else {
		logrus.Error(err)
	}
	data["count"] = runtime.NumCPU()
	return data
}

func (c CPUCollector) getCPUPercentage() map[string]interface{} {
	return map[string]interface{}{}
}

func (c CPUDataGetter) GetCPULoadAverage() map[string]interface{} {
	return map[string]interface{}{}
}

func (c CPUDataGetter) getCPUInfoData() []string {
	return []string{}
}
