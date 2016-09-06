package hostInfo

type IopsCollector struct {
	GOOS string
}

func (i IopsCollector) GetData() (map[string]interface{}, error) {
	return i.parseIopsData()
}

func (i IopsCollector) KeyName() string {
	return "iopsInfo"
}

func (i IopsCollector) GetLabels(prefix string) (map[string]string, error) {
	return map[string]string{}, nil
}
