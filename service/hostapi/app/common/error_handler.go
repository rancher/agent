package common

import (
	"fmt"
	"net/http"
)

type ErrorHandler func(http.ResponseWriter, *http.Request) error

func (fn ErrorHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	err := fn(rw, req)
	if err != nil {
		CheckError(err, 2)
		http.Error(rw, fmt.Sprintf("ERROR: %s", err), 500)
	}
}
