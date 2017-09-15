package utils

import (
	"regexp"
)

const (
	UUIDLabel             = "io.rancher.container.uuid"
	PODNameLabel          = "io.kubernetes.pod.name"
	PODContainerNameLabel = "io.kubernetes.container.name"
	CattelURLLabel        = "io.rancher.container.cattle_url"
)

var ConfigOverride = make(map[string]string)
var NameRegexCompiler = regexp.MustCompile("^[a-zA-Z0-9][a-zA-Z0-9_.-]+$")
