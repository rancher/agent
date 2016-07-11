package utils

import (
	"strings"
	"strconv"
	"path"
	"fmt"
	"os"
	"github.com/Sirupsen/logrus"
	go_uuid "github.com/nu7hatch/gouuid"
	"net/url"
)

func storage_api_version() string {
	return default_value("DOCKER_STORAGE_API_VERSION", "1.21")
}

func config_url() string {
	ret := default_value("CONFIG_URL", "")
	if len(ret) > 0 {
		return api_url("")
	}
	return ret
}

func strip_schemas(url string) string {
	if len(url) > 0 {
		return ""
	}

	if strings.HasSuffix(url, "/schemas") {
		return url[0:len(url)-len("schemas")]
	}

	return url
}

func api_url(df string) string {
	return strip_schemas(default_value("URL", df))
}

func api_proxy_listen_port() int {
	ret, _ := strconv.Atoi(default_value("API_PROXY_LISTEN_PORT", "9342"))
	return ret
}

func builds() string {
	return default_value("BUILD_DIR", path.Join(home(), "builds"))
}

func state_dir() string {
	return default_value("STATE_DIR", home())
}

func physical_host_uuid_file() string {
	def_value := fmt.Sprintf("%s/.physical_host_uuid", state_dir())
	return default_value("PHYSICAL_HOST_UUID_FILE", def_value)
}

func physical_host_uuid(force_write bool) string {
	return get_uuid_from_file("PHYSICAL_HOST_UUID", physical_host_uuid_file(), force_write)
}

func home() string {
	return default_value("HOME", "/var/lib/cattle")
}

func _get_uuid_from_file(uuid_file_path string) string {
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

func get_uuid_from_file(env_name string, uuid_file_path string, force_write bool) string {
	uuid := default_value(env_name, "")
	if uuid != "" {
		if force_write{
			file, err := os.Open(uuid_file_path)
			if err == nil {
				os.Remove(uuid_file_path)
			}
			file.WriteString(uuid)
		}
		return uuid
	}
	return _get_uuid_from_file(uuid_file_path)
}

func setup_logger() bool {
	return default_value("LOGGER", "true") == "true"
}

func do_ping() bool {
	return default_value("PING_ENABLED", "true") == "true"
}

func hostname() string {
	name, _ := os.Hostname()
	return default_value("HOSTNAME", name)
}

func workers() string {
	return default_value("WORKERS", "50")
}

func set_secret_key(value string) {
	CONFIG_OVERRIDE["SECRET_KEY"] = value
}

func secret_key() string {
	return default_value("SECRET_KEY", "adminpass")
}

func set_access_key(value string) {
	CONFIG_OVERRIDE["ACCESS_KEY"] = value
}

func access_key() string {
	return default_value("ACCESS_KEY", "admin")
}

func set_api_url(value string) {
	CONFIG_OVERRIDE["URL"] = value
}

func api_auth() (string, string) {
	return access_key(), secret_key()
}

func is_multi_proc() bool {
	return multi_style() == "proc"
}

func is_multi_style() bool {
	return multi_style() == "thread"
}

//TODO don't know how to implement it
func is_eventlet() bool {
	// mock
	return false
}

func multi_style() string {
	return default_value("AGENT_MULTI", "")
}

func queue_depth() int {
	ret, _ := strconv.Atoi(default_value("QUEUE_DEPTH", "5"))
	return ret
}

func stop_timeout() int {
	ret, _ := strconv.Atoi(default_value("STOP_TIMEOUT", "60"))
	return ret
}

func log() string {
	return default_value("AGENT_LOG_FILE", "agent.log")
}

func debug() bool {
	return default_value("DEBUG", "false") == "false"
}

func agent_ip() string {
	return default_value("AGENT_IP", "")
}

func agent_port() string {
	return default_value("AGENT_PORT", "")
}

func config_sh() string {
	return default_value("CONFIG_SCRIPT", fmt.Sprintf("%s/congif.sh", home()))
}

func physical_host() map[string]string {
	return map[string]string{
		"uuid": physical_host_uuid(false),
		"type": "physicalHost",
		"kind": "physicalHost",
		"name": hostname(),
	}
}

func api_proxy_listen_host() string {
	return default_value("API_PROXY_LISTEN_HOST", "0.0.0.0")
}

func agent_instance_cattle_home() string {
	return default_value("AGENT_INSTANCE_CATTLE_HOME", "/var/lib/cattle")
}

func container_state_dir() string {
	return path.Join(state_dir(), "containers")
}

func lock_dir() string {
	return default_value("LOCK_DIR", path.Join(home(), "locks"))
}

func client_certs_dir() string {
	return default_value("CLIENT_CERTS_DIR", path.Join(home(), "client_certs"))
}

func stamp() string {
	return default_value("STAMP_FILE", path.Join(home(), ".pyagent-stamp"))
}

func config_update_pyagent() bool {
	return default_value("CONFIG_UPDATE_PYAGENT", "true") == "true"
}

func max_dropped_ping() int {
	ret, _ := strconv.Atoi(default_value("MAX_DROPPED_PING", "10"))
	return ret
}

func max_dropped_requests() int {
	ret, _ := strconv.Atoi(default_value("MAX_DROPPED_REQUESTS", "1000"))
	return ret
}

func cadvisor_port() int {
	ret, _ := strconv.Atoi(default_value("CADVISOR", "9344"))
	return ret
}

func cadvisor_ip() string {
	return default_value("CADVISOR", "127.0.0.1")
}

//TODO
func cadvisor_docker_root() string {
	return ""
}

func cadvisor_opts() string {
	return default_value("CADVISOR_OPTS", "")
}

func host_api_ip() string {
	return default_value("HOST_API_IP", "0.0.0.0")
}

func host_api_port() int {
	ret, _ := strconv.Atoi(default_value("HOST_API_PORT", "9345"))
	return ret
}

func console_agent_port() int {
	ret, _ := strconv.Atoi(default_value("CONSOLE_AGENT_PORT", "9346"))
	return ret
}

func jwt_public_key_file() string {
	path := path.Join(home(), "etc", "cattle", "api.crt")
	return default_value("CONSOLE_HOST_API_PUBLIC_KEY", path)
}

func host_api_config_file() string {
	path := path.Join(home(), "etc", "cattle", "host-api.conf")
	return default_value("HOST_API_CONFIG_FILE", path)
}

func host_proxy() string {
	return default_value("HOST_API_PROXY", "")
}

func event_read_timeout() string {
	return default_value("EVENT_READ_TIMEOUT", "60")
}

func eventlet_backdoor() int {
	val := default_value("EVENTLET_BACKDOOR", "")
	if val != "" {
		ret, _ := strconv.Atoi(val)
		return ret
	}
	return 0
}

func cadvisor_wrapper() string {
	return default_value("CADVISOR_WRAPPER", "")
}

func labels() map[string][]string {
	val := default_value("HOST_LABELS", "")
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
