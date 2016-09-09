package constants

import (
	"regexp"
)

const SystemLabels = "io.rancher.container.system"
const ContainerNameLabel = "io.rancher.container.name"
const PullImageLabels = "io.rancher.container.pull_image"
const UUIDLabel = "io.rancher.container.uuid"
const CattelURLLabel = "io.rancher.container.cattle_url"
const AgentIDLabel = "io.rancher.container.agent_id"

const DefaultVersion = "1.22"
const TempName = "work"
const TempPrefix = "cattle-temp-"

var ConfigOverride = make(map[string]string)
var HTTPProxyList = []string{"http_proxy", "https_proxy", "NO_PROXY"}
var NameRegexCompiler = regexp.MustCompile("^[a-zA-Z0-9][a-zA-Z0-9_.-]+$")
