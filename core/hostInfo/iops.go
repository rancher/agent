package hostInfo

import "github.com/pkg/errors"

type IopsCollector struct {
}

func (i IopsCollector) GetData() (map[string]interface{}, error) {
	data, err := i.parseIopsData()
	if err != nil {
		return map[string]interface{}{}, errors.WithStack(err)
	}
	return data, nil
}

func (i IopsCollector) KeyName() string {
	return "iopsInfo"
}

func (i IopsCollector) GetLabels(prefix string) (map[string]string, error) {
	return map[string]string{}, nil
}
