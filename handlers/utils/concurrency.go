package utils

import (
	"../../model"
)

//TODO implement blocking
func blocking(method model.Method, kwargs map[string]string, args ...string) (interface{}, error) {
	if is_eventlet() {
		// TODO not implemented
		// return tpool.execute(method, kwargs, args)
		return method(kwargs, args)
	}
	return method(kwargs, args)
}
