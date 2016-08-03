package utils

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	goUUID "github.com/nu7hatch/gouuid"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"github.com/rancher/agent/handlers/docker"
	"golang.org/x/net/context"
)

func storageAPIVersion() string {
	return defaultValue("DOCKER_STORAGE_API_VERSION", "1.21")
}

func ConfigURL() string {
	ret := defaultValue("CONFIG_URL", "")
	if len(ret) == 0 {
		return ApiURL("")
	}
	return ret
}

func stripSchemas(url string) string {
	if len(url) > 0 {
		return ""
	}

	if strings.HasSuffix(url, "/schemas") {
		return url[0 : len(url)-len("schemas")]
	}

	return url
}

func ApiURL(df string) string {
	return stripSchemas(defaultValue("URL", df))
}

func apiProxyListenPort() int {
	ret, _ := strconv.Atoi(defaultValue("API_PROXY_LISTEN_PORT", "9342"))
	return ret
}

func builds() string {
	return defaultValue("BUILD_DIR", path.Join(Home(), "builds"))
}

func stateDir() string {
	return defaultValue("STATE_DIR", Home())
}

func physicalHostUUIDFile() string {
	defValue := fmt.Sprintf("%s/.physical_host_uuid", stateDir())
	return defaultValue("PHYSICAL_HOST_UUID_FILE", defValue)
}

func physicalHostUUID(forceWrite bool) string {
	return GetUUIDFromFile("PHYSICAL_HOST_UUID", physicalHostUUIDFile(), forceWrite)
}

func Home() string {
	return defaultValue("HOME", "/var/lib/cattle")
}

func getUUIDFromFile(uuidFilePath string) string {
	uuid := ""

	fileReader, err := os.Open(uuidFilePath)
	if err != nil {
		logrus.Error(err)
	} else {
		uuid = ReadBuffer(fileReader)
	}
	if uuid == "" {
		newUUID, err1 := goUUID.NewV4()
		if err1 != nil {
			logrus.Error(err)
		} else {
			uuid = newUUID.String()
		}
		file, _ := os.Create(uuidFilePath)
		file.WriteString(uuid)
	}
	return uuid
}

func GetUUIDFromFile(envName string, uuidFilePath string, forceWrite bool) string {
	uuid := defaultValue(envName, "")
	if uuid != "" {
		if forceWrite {
			file, err := os.Open(uuidFilePath)
			if err == nil {
				os.Remove(uuidFilePath)
			}
			file.WriteString(uuid)
		}
		return uuid
	}
	return getUUIDFromFile(uuidFilePath)
}

func setupLogger() bool {
	return defaultValue("LOGGER", "true") == "true"
}

func DoPing() bool {
	return defaultValue("PING_ENABLED", "true") == "true"
}

func hostname() string {
	name, _ := os.Hostname()
	return defaultValue("HOSTNAME", name)
}

func workers() string {
	return defaultValue("WORKERS", "50")
}

func setSecretKey(value string) {
	ConfigOverride["SECRET_KEY"] = value
}

func SecretKey() string {
	return defaultValue("SECRET_KEY", "adminpass")
}

func setAccessKey(value string) {
	ConfigOverride["ACCESS_KEY"] = value
}

func AccessKey() string {
	return defaultValue("ACCESS_KEY", "admin")
}

func setAPIURL(value string) {
	ConfigOverride["URL"] = value
}

func apiAuth() (string, string) {
	return AccessKey(), SecretKey()
}

func isMultiProc() bool {
	return multiStyle() == "proc"
}

func isMultiStyle() bool {
	return multiStyle() == "thread"
}

//TODO don't know how to implement it
func isEventlet() bool {
	// mock
	return false
}

func multiStyle() string {
	return defaultValue("AGENT_MULTI", "")
}

func queueDepth() int {
	ret, _ := strconv.Atoi(defaultValue("queueDepth", "5"))
	return ret
}

func stopTimeout() int {
	ret, _ := strconv.Atoi(defaultValue("stopTimeout", "60"))
	return ret
}

func log() string {
	return defaultValue("AGENT_LOG_FILE", "agent.log")
}

func debug() bool {
	return defaultValue("DEBUG", "false") == "false"
}

func agentIP() string {
	return defaultValue("AGENT_IP", "")
}

func agentPort() string {
	return defaultValue("agentPort", "")
}

func ConfigSh() string {
	return defaultValue("CONFIG_SCRIPT", fmt.Sprintf("%s/config.sh", Home()))
}

func physicalHost() map[string]interface{} {
	return map[string]interface{}{
		"uuid": physicalHostUUID(false),
		"type": "physicalHost",
		"kind": "physicalHost",
		"name": hostname(),
	}
}

func apiProxyListenHost() string {
	return defaultValue("apiProxyListenHost", "0.0.0.0")
}

func agentInstanceCattleHome() string {
	return defaultValue("agentInstanceCattleHome", "/var/lib/cattle")
}

func ContainerStateDir() string {
	return path.Join(stateDir(), "containers")
}

func lockDir() string {
	return defaultValue("lockDir", path.Join(Home(), "locks"))
}

func clientCertsDir() string {
	return defaultValue("CLIENT_CERTS_DIR", path.Join(Home(), "client_certs"))
}

func stamp() string {
	return defaultValue("STAMP_FILE", path.Join(Home(), ".pyagent-stamp"))
}

func ConfigUpdatePyagent() bool {
	return defaultValue("CONFIG_UPDATE_PYAGENT", "true") == "true"
}

func maxDroppedPing() int {
	ret, _ := strconv.Atoi(defaultValue("maxDroppedPing", "10"))
	return ret
}

func maxDroppedRequests() int {
	ret, _ := strconv.Atoi(defaultValue("maxDroppedRequests", "1000"))
	return ret
}

func CadvisorDockerRoot() string {
	info, _ := docker.GetClient(DefaultVersion).Info(context.Background())
	return info.DockerRootDir
}

func CadvisorOpts() string {
	return defaultValue("CADVISOR_OPTS", "")
}

func HostAPIIP() string {
	return defaultValue("HOST_API_IP", "0.0.0.0")
}

func HostAPIPort() string {
	return defaultValue("HOST_API_PORT", "9345")
}

func consoleAgentPort() int {
	ret, _ := strconv.Atoi(defaultValue("CONSOLE_AGENT_PORT", "9346"))
	return ret
}

func JwtPublicKeyFile() string {
	path := path.Join(Home(), "etc", "cattle", "api.crt")
	return defaultValue("CONSOLE_HOST_API_PUBLIC_KEY", path)
}

func HostAPIConfigFile() string {
	path := path.Join(Home(), "etc", "cattle", "host-api.conf")
	return defaultValue("HOST_API_CONFIG_FILE", path)
}

func hostProxy() string {
	return defaultValue("HOST_API_PROXY", "")
}

func eventReadTimeout() string {
	return defaultValue("EVENT_READ_TIMEOUT", "60")
}

func eventletBackdoor() int {
	val := defaultValue("eventletBackdoor", "")
	if val != "" {
		ret, _ := strconv.Atoi(val)
		return ret
	}
	return 0
}

func CadvisorWrapper() string {
	return defaultValue("CADVISOR_WRAPPER", "")
}

func labels() map[string][]string {
	val := defaultValue("HOST_LABELS", "")
	if val != "" {
		m, err := url.ParseQuery(val)
		if err != nil {
			logrus.Error(err)
		} else {
			return m
		}
	}
	return map[string][]string{}
}

func DockerEnable() bool {
	return defaultValue("DOCKER_ENABLED", "true") == "true"
}

func DockerHostIP() string {
	return defaultValue("DOCKER_HOST_IP", agentIP())
}

func DockerUUID() string {
	return GetUUIDFromFile("DOCKER_UUID", dockerUUIDFile(), false)
}

func dockerUUIDFile() string {
	defValue := fmt.Sprintf("%v/.docker_uuid", stateDir())
	return defaultValue("DOCKER_UUID_FILE", defValue)
}

func CadvisorIP() string {
	return defaultValue("CADVISOR_IP", "127.0.0.1")
}

func CadvisorPort() string {
	return defaultValue("CADVISOR_PORT", "9344")
}

func CadvisorInterval() string {
	return defaultValue("CADVISOR_INTERVAL", "1s")
}
