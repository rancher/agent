package constants

import "regexp"

const SystemLables = "io.rancher.container.system"
const UUIDLabel = "io.rancher.container.uuid"
const DefaultVersion = "1.22"
const TempName = "work"
const TempPrefix = "cattle-temp-"

var ConfigOverride = make(map[string]string)

// rename to something like HostnameRegexp
var RegexCompiler = regexp.MustCompile("^[a-zA-Z0-9][a-zA-Z0-9_.-]+$")
