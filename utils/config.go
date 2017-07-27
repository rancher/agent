package utils

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"

	"github.com/Sirupsen/logrus"
)

func URL() string {
	ret := DefaultValue("CONFIG_URL", "")
	if len(ret) == 0 {
		return APIURL("")
	}
	return ret
}

func stripSchemas(url string) string {
	if len(url) == 0 {
		return ""
	}

	if strings.HasSuffix(url, "/schemas") {
		return url[0 : len(url)-len("schemas")]
	}

	return url
}

func APIURL(df string) string {
	return stripSchemas(DefaultValue("URL", df))
}

func APIProxyListenPort() int {
	ret, _ := strconv.Atoi(DefaultValue("API_PROXY_LISTEN_PORT", "9342"))
	return ret
}

func Builds() string {
	return DefaultValue("BUILD_DIR", path.Join(Home(), "builds"))
}

func StateDir() string {
	return DefaultValue("STATE_DIR", Home())
}

func KeyFile() string {
	defValue := fmt.Sprintf("%s/../etc/ssl/host.key", StateDir())
	return DefaultValue("HOST_KEY_FILE", defValue)
}

func Home() string {
	if runtime.GOOS == "windows" {
		return DefaultValue("HOME", "c:/ProgramData/rancher")
	}
	return DefaultValue("HOME", "/var/lib/cattle")
}

func DoPing() bool {
	return DefaultValue("PING_ENABLED", "true") == "true"
}

func SetSecretKey(value string) {
	ConfigOverride["SECRET_KEY"] = value
}

func SecretKey() string {
	return DefaultValue("SECRET_KEY", "adminpass")
}

func SetAccessKey(value string) {
	ConfigOverride["ACCESS_KEY"] = value
}

func AccessKey() string {
	return DefaultValue("ACCESS_KEY", "admin")
}

func SetAPIURL(value string) {
	ConfigOverride["URL"] = value
}

func agentIP() string {
	return DefaultValue("AGENT_IP", "")
}

func Sh() string {
	return DefaultValue("CONFIG_SCRIPT", fmt.Sprintf("%s/config.sh", Home()))
}

func UpdatePyagent() bool {
	return DefaultValue("CONFIG_UPDATE_PYAGENT", "true") == "true"
}

func HostAPIIP() string {
	return DefaultValue("HOST_API_IP", "0.0.0.0")
}

func HostAPIPort() string {
	return DefaultValue("HOST_API_PORT", "9345")
}

func JwtPublicKeyFile() string {
	path := path.Join(Home(), "etc", "cattle", "api.crt")
	return DefaultValue("CONSOLE_HOST_API_PUBLIC_KEY", path)
}

func HostProxy() string {
	return DefaultValue("HOST_API_PROXY", "")
}

func Labels() map[string]string {
	val := DefaultValue("HOST_LABELS", "")
	ret := map[string]string{}
	if val != "" {
		m, err := url.ParseQuery(val)
		if err != nil {
			logrus.Error(err)
		}
		for k, v := range m {
			ret[k] = v[0]
		}
	}
	return ret
}

func DockerEnable() bool {
	return DefaultValue("DOCKER_ENABLED", "true") == "true"
}

func DockerHostIP() string {
	return DefaultValue("DOCKER_HOST_IP", agentIP())
}

func DefaultValue(name string, df string) string {
	if value, ok := ConfigOverride[name]; ok {
		return value
	}
	if result := os.Getenv(fmt.Sprintf("CATTLE_%s", name)); result != "" {
		return result
	}
	return df
}

func Stamp() string {
	return DefaultValue("STAMP_FILE", "/usr/bin/agent")
}
