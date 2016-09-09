package config

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	goUUID "github.com/nu7hatch/gouuid"
	"github.com/pkg/errors"
	"github.com/rancher/agent/model"
	"github.com/rancher/agent/utilities/constants"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
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

func PhysicalHostUUID(forceWrite bool) (string, error) {
	return GetUUIDFromFile("PHYSICAL_HOST_UUID", physicalHostUUIDFile(), forceWrite)
}

func Home() string {
	if runtime.GOOS == "windows" {
		return DefaultValue("HOME", "c:/Cattle")
	}
	return DefaultValue("HOME", "/var/lib/cattle")
}

func getUUIDFromFile(uuidFilePath string) (string, error) {
	uuid := ""

	fileBuffer, err := ioutil.ReadFile(uuidFilePath)
	if err != nil && !os.IsNotExist(err) {
		return "", errors.Wrap(err, constants.ReadUUIDFromFileError)
	}
	uuid = string(fileBuffer)
	if uuid == "" {
		newUUID, err := goUUID.NewV4()
		if err != nil {
			return "", errors.Wrap(err, constants.ReadUUIDFromFileError)
		}
		uuid = newUUID.String()
		file, err := os.Create(uuidFilePath)
		if err != nil {
			return "", errors.Wrap(err, constants.ReadUUIDFromFileError)
		}
		if _, err := file.WriteString(uuid); err != nil {
			return "", errors.Wrap(err, constants.ReadUUIDFromFileError)
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
				return "", errors.Wrap(err, constants.GetUUIDFromFileError)
			}
			file, err := os.Create(uuidFilePath)
			if err != nil {
				return "", errors.Wrap(err, constants.GetUUIDFromFileError)
			}
			if _, err := file.WriteString(uuid); err != nil {
				return "", errors.Wrap(err, constants.GetUUIDFromFileError)
			}
		}
		return uuid, nil
	}
	return getUUIDFromFile(uuidFilePath)
}

func DoPing() bool {
	return DefaultValue("PING_ENABLED", "true") == "true"
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
		return model.PingResource{}, errors.Wrap(err, constants.PhysicalHostError)
	}
	hostname, err := Hostname()
	if err != nil {
		return model.PingResource{}, errors.Wrap(err, constants.PhysicalHostError)
	}
	return model.PingResource{
		UUID: uuid,
		Type: "physicalHost",
		Kind: "physicalHost",
		Name: hostname,
	}, nil
}

func ContainerStateDir() string {
	return path.Join(StateDir(), "containers")
}

func UpdatePyagent() bool {
	return DefaultValue("CONFIG_UPDATE_PYAGENT", "true") == "true"
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

func JwtPublicKeyFile() string {
	path := path.Join(Home(), "etc", "cattle", "api.crt")
	return DefaultValue("CONSOLE_HOST_API_PUBLIC_KEY", path)
}

func HostProxy() string {
	return DefaultValue("HOST_API_PROXY", "")
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

func Stamp() string {
	return DefaultValue("STAMP_FILE", path.Join(Home(), ".pyagent-stamp"))
}
