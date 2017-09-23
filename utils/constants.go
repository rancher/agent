package utils

import (
	"regexp"
)

const (
	UUIDLabel = "io.rancher.container.agent.uuid"
)

var ConfigOverride = make(map[string]string)
var NameRegexCompiler = regexp.MustCompile("^[a-zA-Z0-9][a-zA-Z0-9_.-]+$")
