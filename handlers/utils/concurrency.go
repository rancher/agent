package utils

import (
	"github.com/rancher/agent/model"
)

//TODO implement blocking
func blocking(method model.Method, kwargs map[string]string, args ...string) (interface{}, error) {
	if isEventlet() {
		// TODO not implemented
		// return tpool.execute(method, kwargs, args)
		return method(kwargs, args)
	}
	return method(kwargs, args)
}
