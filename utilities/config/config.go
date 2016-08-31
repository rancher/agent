package config

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strconv"
	"strings"

	"github.com/Sirupsen/logrus"
	goUUID "github.com/nu7hatch/gouuid"
	"github.com/rancher/agent/utilities/constants"
)

func StorageAPIVersion() string {
	return DefaultValue("DOCKER_STORAGE_API_VERSION", "1.21")
}

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

func physicalHostUUIDFile() string {
	defValue := fmt.Sprintf("%s/.physical_host_uuid", StateDir())
	return DefaultValue("PHYSICAL_HOST_UUID_FILE", defValue)
}

func PhysicalHostUUID(forceWrite bool) string {
	return GetUUIDFromFile("PHYSICAL_HOST_UUID", physicalHostUUIDFile(), forceWrite)
}

func Home() string {
	if runtime.GOOS == "windows" {
		return DefaultValue("HOME", "c:/Cattle")
	}
	return DefaultValue("HOME", "/var/lib/cattle")
}

func getUUIDFromFile(uuidFilePath string) string {
	uuid := ""

	if fileBuffer, err := ioutil.ReadFile(uuidFilePath); err == nil {
		uuid = string(fileBuffer)
	}
	if uuid == "" {
		if newUUID, err := goUUID.NewV4(); err == nil {
			// if err != nil, panic(err)
			uuid = newUUID.String()
			file, _ := os.Create(uuidFilePath)
			file.WriteString(uuid)
		}
	}
	return uuid
}

func GetUUIDFromFile(envName string, uuidFilePath string, forceWrite bool) string {
	uuid := DefaultValue(envName, "")
	if uuid != "" {
		if forceWrite {
			_, err := os.Open(uuidFilePath)
			if err == nil {
				os.Remove(uuidFilePath)
			}
			file, _ := os.Create(uuidFilePath)
			file.WriteString(uuid)
		}
		return uuid
	}
	return getUUIDFromFile(uuidFilePath)
}

func setupLogger() bool {
	return DefaultValue("LOGGER", "true") == "true"
}

func DoPing() bool {
	return DefaultValue("PING_ENABLED", "true") == "true"
}

func Hostname() string {
	var hostname string
	if runtime.GOOS == "windows" {
		hostname, _ = os.Hostname()
	} else {
		hostname = getFQDNLinux()
	}
	return DefaultValue("HOSTNAME", hostname)
}

func workers() string {
	return DefaultValue("WORKERS", "50")
}

func SetSecretKey(value string) {
	constants.ConfigOverride["SECRET_KEY"] = value
}

func SecretKey() string {
	return DefaultValue("SECRET_KEY", "adminpass")
}

func SetAccessKey(value string) {
	constants.ConfigOverride["ACCESS_KEY"] = value
}

func AccessKey() string {
	return DefaultValue("ACCESS_KEY", "admin")
}

func SetAPIURL(value string) {
	constants.ConfigOverride["URL"] = value
}

func apiAuth() (string, string) {
	return AccessKey(), SecretKey()
}

func isMultiProc() bool {
	return multiStyle() == "proc"
}

// TODO: check if all functions here are used
func isMultiStyle() bool {
	return multiStyle() == "thread"
}

//TODO don't know how to implement it
func isEventlet() bool {
	// mock
	return false
}

func multiStyle() string {
	return DefaultValue("AGENT_MULTI", "")
}

func queueDepth() int {
	ret, _ := strconv.Atoi(DefaultValue("queueDepth", "5"))
	return ret
}

func stopTimeout() int {
	ret, _ := strconv.Atoi(DefaultValue("stopTimeout", "60"))
	return ret
}

func log() string {
	return DefaultValue("AGENT_LOG_FILE", "agent.log")
}

func debug() bool {
	return DefaultValue("DEBUG", "false") == "false"
}

func agentIP() string {
	return DefaultValue("AGENT_IP", "")
}

func agentPort() string {
	return DefaultValue("agentPort", "")
}

func Sh() string {
	return DefaultValue("CONFIG_SCRIPT", fmt.Sprintf("%s/config.sh", Home()))
}

func PhysicalHost() map[string]interface{} {
	return map[string]interface{}{
		"uuid": PhysicalHostUUID(false),
		"type": "physicalHost",
		"kind": "physicalHost",
		"name": Hostname(),
	}
}

func APIProxyListenHost() string {
	return DefaultValue("API_PROXY_LISTEN_HOST", "0.0.0.0")
}

func agentInstanceCattleHome() string {
	return DefaultValue("agentInstanceCattleHome", "/var/lib/cattle")
}

func ContainerStateDir() string {
	return path.Join(StateDir(), "containers")
}

func lockDir() string {
	return DefaultValue("lockDir", path.Join(Home(), "locks"))
}

func clientCertsDir() string {
	return DefaultValue("CLIENT_CERTS_DIR", path.Join(Home(), "client_certs"))
}

func stamp() string {
	return DefaultValue("STAMP_FILE", path.Join(Home(), ".pyagent-stamp"))
}

func UpdatePyagent() bool {
	return DefaultValue("CONFIG_UPDATE_PYAGENT", "true") == "true"
}

func maxDroppedPing() int {
	ret, _ := strconv.Atoi(DefaultValue("maxDroppedPing", "10"))
	return ret
}

func maxDroppedRequests() int {
	ret, _ := strconv.Atoi(DefaultValue("maxDroppedRequests", "1000"))
	return ret
}

func CadvisorOpts() string {
	return DefaultValue("CADVISOR_OPTS", "")
}

func HostAPIIP() string {
	return DefaultValue("HOST_API_IP", "0.0.0.0")
}

func HostAPIPort() string {
	return DefaultValue("HOST_API_PORT", "9345")
}

func consoleAgentPort() int {
	ret, _ := strconv.Atoi(DefaultValue("CONSOLE_AGENT_PORT", "9346"))
	return ret
}

func JwtPublicKeyFile() string {
	path := path.Join(Home(), "etc", "cattle", "api.crt")
	return DefaultValue("CONSOLE_HOST_API_PUBLIC_KEY", path)
}

func HostAPIConfigFile() string {
	path := path.Join(Home(), "etc", "cattle", "host-api.conf")
	return DefaultValue("HOST_API_CONFIG_FILE", path)
}

func HostProxy() string {
	return DefaultValue("HOST_API_PROXY", "")
}

func eventReadTimeout() string {
	return DefaultValue("EVENT_READ_TIMEOUT", "60")
}

func eventletBackdoor() int {
	val := DefaultValue("eventletBackdoor", "")
	if val != "" {
		ret, _ := strconv.Atoi(val)
		return ret
	}
	return 0
}

func CadvisorWrapper() string {
	return DefaultValue("CADVISOR_WRAPPER", "")
}

func Labels() map[string][]string {
	val := DefaultValue("HOST_LABELS", "")
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
	return DefaultValue("DOCKER_ENABLED", "true") == "true"
}

func DockerHostIP() string {
	return DefaultValue("DOCKER_HOST_IP", agentIP())
}

func DockerUUID() string {
	return GetUUIDFromFile("DOCKER_UUID", dockerUUIDFile(), false)
}

func dockerUUIDFile() string {
	defValue := fmt.Sprintf("%v/.docker_uuid", StateDir())
	return DefaultValue("DOCKER_UUID_FILE", defValue)
}

func CadvisorIP() string {
	return DefaultValue("CADVISOR_IP", "127.0.0.1")
}

func CadvisorPort() string {
	return DefaultValue("CADVISOR_PORT", "9344")
}

func CadvisorInterval() string {
	return DefaultValue("CADVISOR_INTERVAL", "1s")
}

func DefaultValue(name string, df string) string {
	if value, ok := constants.ConfigOverride[name]; ok {
		return value
	}
	if result := os.Getenv(fmt.Sprintf("CATTLE_%s", name)); result != "" {
		return result
	}
	return df
}

// handle error, maybe panic
func getFQDNLinux() string {
	cmd := exec.Command("/bin/hostname", "-f")
	var out bytes.Buffer
	cmd.Stdout = &out
	//TODO: use cmd.CombinedOutput()
	cmd.CombinedOutput()
	err := cmd.Run()
	if err != nil {
		return ""
	}
	fqdn := out.String()
	fqdn = fqdn[:len(fqdn)-1]
	return fqdn
}
