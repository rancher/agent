package utils

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	go_uuid "github.com/nu7hatch/gouuid"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
)

func storageApiVersion() string {
	return defaultValue("DOCKER_storageApiVersion", "1.21")
}

func configUrl() string {
	ret := defaultValue("configUrl", "")
	if len(ret) > 0 {
		return apiUrl("")
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

func apiUrl(df string) string {
	return stripSchemas(defaultValue("URL", df))
}

func apiProxyListenPort() int {
	ret, _ := strconv.Atoi(defaultValue("apiProxyListenPort", "9342"))
	return ret
}

func builds() string {
	return defaultValue("BUILD_DIR", path.Join(home(), "builds"))
}

func stateDir() string {
	return defaultValue("stateDir", home())
}

func physicalHostUuidFile() string {
	def_value := fmt.Sprintf("%s/.physicalHostUuid", stateDir())
	return defaultValue("physicalHostUuidFile", def_value)
}

func physicalHostUuid(force_write bool) string {
	return GetUuidFromFile("physicalHostUuid", physicalHostUuidFile(), force_write)
}

func home() string {
	return defaultValue("HOME", "/var/lib/cattle")
}

func getUuidFromFile(uuid_file_path string) string {
	uuid := ""

	file_reader, err := os.Open(uuid_file_path)
	if err != nil {
		logrus.Error(err)
	} else {
		uuid = readBuffer(file_reader)
	}
	if uuid == "" {
		new_uuid, err1 := go_uuid.NewV4()
		if err1 != nil {
			logrus.Error(err)
		} else {
			uuid = new_uuid.String()
		}
		file_reader.WriteString(uuid)
	}
	return uuid
}

func GetUuidFromFile(env_name string, uuid_file_path string, force_write bool) string {
	uuid := defaultValue(env_name, "")
	if uuid != "" {
		if force_write {
			file, err := os.Open(uuid_file_path)
			if err == nil {
				os.Remove(uuid_file_path)
			}
			file.WriteString(uuid)
		}
		return uuid
	}
	return getUuidFromFile(uuid_file_path)
}

func setupLogger() bool {
	return defaultValue("LOGGER", "true") == "true"
}

func doPing() bool {
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
	CONFIG_OVERRIDE["SECRET_KEY"] = value
}

func secret_key() string {
	return defaultValue("SECRET_KEY", "adminpass")
}

func setAccessKey(value string) {
	CONFIG_OVERRIDE["accessKey"] = value
}

func accessKey() string {
	return defaultValue("accessKey", "admin")
}

func setApiUrl(value string) {
	CONFIG_OVERRIDE["URL"] = value
}

func apiAuth() (string, string) {
	return accessKey(), secret_key()
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

func agentIp() string {
	return defaultValue("agentIp", "")
}

func agentPort() string {
	return defaultValue("agentPort", "")
}

func configSh() string {
	return defaultValue("CONFIG_SCRIPT", fmt.Sprintf("%s/congif.sh", home()))
}

func physicalHost() map[string]string {
	return map[string]string{
		"uuid": physicalHostUuid(false),
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

func client_certs_dir() string {
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

func cadvisorPort() int {
	ret, _ := strconv.Atoi(defaultValue("CADVISOR", "9344"))
	return ret
}

func cadvisorIp() string {
	return defaultValue("CADVISOR", "127.0.0.1")
}

//TODO
func cadvisorDockerRoot() string {
	return ""
}

func cadvisorOpts() string {
	return defaultValue("cadvisorOpts", "")
}

func hostApiIp() string {
	return defaultValue("hostApiIp", "0.0.0.0")
}

func hostApiPort() int {
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

func hostApiConfigFile() string {
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
