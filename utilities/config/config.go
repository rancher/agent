package config

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"

	"github.com/leodotcloud/log"
	goUUID "github.com/nu7hatch/gouuid"
	"github.com/pkg/errors"
	"github.com/rancher/agent/model"
	"github.com/rancher/agent/utilities/constants"
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

func physicalHostUUIDFile() string {
	defValue := fmt.Sprintf("%s/.physical_host_uuid", StateDir())
	return DefaultValue("PHYSICAL_HOST_UUID_FILE", defValue)
}

func PhysicalHostUUID(forceWrite bool) (string, error) {
	return GetUUIDFromFile("PHYSICAL_HOST_UUID", physicalHostUUIDFile(), forceWrite)
}

func Home() string {
	if runtime.GOOS == "windows" {
		return DefaultValue("HOME", "c:/ProgramData/rancher")
	}
	return DefaultValue("HOME", "/var/lib/cattle")
}

func getUUIDFromFile(uuidFilePath string) (string, error) {
	uuid := ""

	fileBuffer, err := ioutil.ReadFile(uuidFilePath)
	if err != nil && !os.IsNotExist(err) {
		return "", errors.Wrap(err, constants.ReadUUIDFromFileError+"failed to read uuid file")
	}
	uuid = string(fileBuffer)
	if uuid == "" {
		newUUID, err := goUUID.NewV4()
		if err != nil {
			return "", errors.Wrap(err, constants.ReadUUIDFromFileError+"failed to generate uuid")
		}
		uuid = newUUID.String()
		file, err := os.Create(uuidFilePath)
		if err != nil {
			return "", errors.Wrap(err, constants.ReadUUIDFromFileError+"failed to create uuid file")
		}
		if _, err := file.WriteString(uuid); err != nil {
			return "", errors.Wrap(err, constants.ReadUUIDFromFileError+"failed to write uuid to file")
		}
	}
	return uuid, nil
}

func GetUUIDFromFile(envName string, uuidFilePath string, forceWrite bool) (string, error) {
	uuid := DefaultValue(envName, "")
	if uuid != "" {
		if forceWrite {
			_, err := os.Open(uuidFilePath)
			if err == nil {
				os.Remove(uuidFilePath)
			} else if !os.IsNotExist(err) {
				return "", errors.Wrap(err, constants.GetUUIDFromFileError+"failed to open uuid file")
			}
			file, err := os.Create(uuidFilePath)
			if err != nil {
				return "", errors.Wrap(err, constants.GetUUIDFromFileError+"failed to create uuid file")
			}
			if _, err := file.WriteString(uuid); err != nil {
				return "", errors.Wrap(err, constants.GetUUIDFromFileError+"failed to write uuid to file")
			}
		}
		return uuid, nil
	}
	return getUUIDFromFile(uuidFilePath)
}

func DoPing() bool {
	return DefaultValue("PING_ENABLED", "true") == "true"
}

func DetectCloudProvider() bool {
	return DefaultValue("DETECT_CLOUD_PROVIDER", "true") == "true"
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

func agentIP() string {
	return DefaultValue("AGENT_IP", "")
}

func Sh() string {
	return DefaultValue("CONFIG_SCRIPT", fmt.Sprintf("%s/config.sh", Home()))
}

func PhysicalHost() (model.PingResource, error) {
	uuid, err := PhysicalHostUUID(false)
	if err != nil {
		return model.PingResource{}, errors.Wrap(err, constants.PhysicalHostError+"failed to get physical host uuid")
	}
	hostname, err := Hostname()
	if err != nil {
		return model.PingResource{}, errors.Wrap(err, constants.PhysicalHostError+"failed to get hostname")
	}
	return model.PingResource{
		UUID: uuid,
		Type: "physicalHost",
		Kind: "physicalHost",
		Name: hostname,
	}, nil
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
			log.Error(err)
		}
		for k, v := range m {
			ret[strings.TrimSpace(k)] = strings.TrimSpace(v[0])
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

func SetDockerUUID() (string, error) {
	return GetUUIDFromFile("DOCKER_UUID", dockerUUIDFile(), true)
}

func DockerUUID() (string, error) {
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

func DefaultValue(name string, df string) string {
	if value, ok := constants.ConfigOverride[name]; ok {
		return value
	}
	if result := os.Getenv(fmt.Sprintf("CATTLE_%s", name)); result != "" {
		return result
	}
	return df
}

func Stamp() string {
	return DefaultValue("STAMP_FILE", path.Join(Home(), ".pyagent-stamp"))
}
