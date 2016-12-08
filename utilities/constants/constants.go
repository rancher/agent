package constants

import (
	"regexp"
)

const (
	ContainerNameLabel = "io.rancher.container.name"
	PullImageLabels    = "io.rancher.container.pull_image"
	UUIDLabel          = "io.rancher.container.uuid"
	CattelURLLabel     = "io.rancher.container.cattle_url"
	AgentIDLabel       = "io.rancher.container.agent_id"
	RancherAgentImage  = "io.rancher.host.agent_image"
	RancherIPLabel     = "io.rancher.container.ip"

	DefaultVersion = "1.22"
	TempName       = "work"
	TempPrefix     = "cattle-temp-"
)

var ConfigOverride = make(map[string]string)
var HTTPProxyList = []string{"http_proxy", "https_proxy", "NO_PROXY"}
var NameRegexCompiler = regexp.MustCompile("^[a-zA-Z0-9][a-zA-Z0-9_.-]+$")
