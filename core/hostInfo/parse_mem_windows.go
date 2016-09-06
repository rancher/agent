package hostInfo

func (m MemoryCollector) parseMemInfo() map[string]interface{} {
	data := map[string]interface{}{}
	keys := map[string]string{
		"memFree":  "FreePhysicalMemory",
		"memTotal": "TotalVisibleMemorySize",
	}
	for k, v := range keys {
		value, err := getCommandOutput(v)
		if err != nil {
			return data, errors.Wrap(err, constants.ParseMemInfoError)
		} else {
			pattern := "([0-9]+)"
			possibleMemValue := regexp.MustCompile(pattern).FindString(value)
			memValue, _ := strconv.ParseFloat(possibleMemValue, 64)
			data[k] = memValue / 1024
		}
	}
	return data
}

func getCommandOutput(key string) (string, error) {
	command := exec.Command("PowerShell", "wmic", "os", "get", key)
	output, err := command.Output()
	if err == nil {
		ret := strings.Split(string(output), "\n")[1]
		return ret, nil
	}
	return "", err
}
