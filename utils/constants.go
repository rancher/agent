package utils

import (
	"regexp"
)

const (
	UUIDLabel      = "io.rancher.container.uuid"
	CattelURLLabel = "io.rancher.container.cattle_url"
	AgentIDLabel   = "io.rancher.container.agent_id"
	TempName       = "work"
	TempPrefix     = "cattle-temp-"
)

var ConfigOverride = make(map[string]string)
var NameRegexCompiler = regexp.MustCompile("^[a-zA-Z0-9][a-zA-Z0-9_.-]+$")
