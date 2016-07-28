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
)

func storageAPIVersion() string {
	return defaultValue("DOCKER_STORAGE_API_VERSION", "1.21")
}

func configURL() string {
	ret := defaultValue("CONFIG_URL", "")
	if len(ret) > 0 {
		return apiURL("")
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

func apiURL(df string) string {
	return stripSchemas(defaultValue("URL", df))
}

func apiProxyListenPort() int {
	ret, _ := strconv.Atoi(defaultValue("API_PROXY_LISTEN_PORT", "9342"))
	return ret
}

func builds() string {
	return defaultValue("BUILD_DIR", path.Join(home(), "builds"))
}

func stateDir() string {
	return defaultValue("STATE_DIR", home())
}

func physicalHostUUIDFile() string {
	defValue := fmt.Sprintf("%s/.physical_host_uuid", stateDir())
	return defaultValue("PHYSICAL_HOST_UUID_FILE", defValue)
}

func physicalHostUUID(forceWrite bool) string {
	return GetUUIDFromFile("PHYSICAL_HOST_UUID", physicalHostUUIDFile(), forceWrite)
}

func home() string {
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
		fileReader.WriteString(uuid)
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

func secretKey() string {
	return defaultValue("SECRET_KEY", "adminpass")
}

func setAccessKey(value string) {
	ConfigOverride["accessKey"] = value
}

func accessKey() string {
	return defaultValue("accessKey", "admin")
}

func setAPIURL(value string) {
	ConfigOverride["URL"] = value
}

func apiAuth() (string, string) {
	return accessKey(), secretKey()
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
	return defaultValue("agentIp", "")
}

func agentPort() string {
	return defaultValue("agentPort", "")
}

func configSh() string {
	return defaultValue("CONFIG_SCRIPT", fmt.Sprintf("%s/congif.sh", home()))
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

func containerStateDir() string {
	return path.Join(stateDir(), "containers")
}

func lockDir() string {
	return defaultValue("lockDir", path.Join(home(), "locks"))
}

func clientCertsDir() string {
	return defaultValue("CLIENT_CERTS_DIR", path.Join(home(), "client_certs"))
}

func stamp() string {
	return defaultValue("STAMP_FILE", path.Join(home(), ".pyagent-stamp"))
}

func configUpdatePyagent() bool {
	return defaultValue("configUpdatePyagent", "true") == "true"
}

func maxDroppedPing() int {
	ret, _ := strconv.Atoi(defaultValue("maxDroppedPing", "10"))
	return ret
}

func maxDroppedRequests() int {
	ret, _ := strconv.Atoi(defaultValue("maxDroppedRequests", "1000"))
	return ret
}

//TODO
func cadvisorDockerRoot() string {
	return ""
}

func cadvisorOpts() string {
	return defaultValue("cadvisorOpts", "")
}

func hostAPIIP() string {
	return defaultValue("hostApiIp", "0.0.0.0")
}

func hostAPIPort() int {
	ret, _ := strconv.Atoi(defaultValue("hostApiPort", "9345"))
	return ret
}

func consoleAgentPort() int {
	ret, _ := strconv.Atoi(defaultValue("consoleAgentPort", "9346"))
	return ret
}

func jwtPublicKeyFile() string {
	path := path.Join(home(), "etc", "cattle", "api.crt")
	return defaultValue("CONSOLE_HOST_API_PUBLIC_KEY", path)
}

func hostAPIConfigFile() string {
	path := path.Join(home(), "etc", "cattle", "host-api.conf")
	return defaultValue("hostApiConfigFile", path)
}

func hostProxy() string {
	return defaultValue("HOST_API_PROXY", "")
}

func eventReadTimeout() string {
	return defaultValue("eventReadTimeout", "60")
}

func eventletBackdoor() int {
	val := defaultValue("eventletBackdoor", "")
	if val != "" {
		ret, _ := strconv.Atoi(val)
		return ret
	}
	return 0
}

func cadvisorWrapper() string {
	return defaultValue("cadvisorWrapper", "")
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

func dockerUUID() string {
	return GetUUIDFromFile("DOCKER_UUID", dockerUUIDFile(), false)
}

func dockerUUIDFile() string {
	defValue := fmt.Sprintf("%v/.docker_uuid", stateDir())
	return defaultValue("DOCKER_UUID_FILE", defValue)
}
