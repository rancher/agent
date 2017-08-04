package config

import (
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/golang/glog"
	"github.com/rancher/agent/utils"
)

type config struct {
	DockerURL       string
	Systemd         bool
	NumStats        int
	Auth            bool
	HaProxyMonitor  bool
	Key             string
	HostUUID        string
	Port            int
	IP              string
	ParsedPublicKey interface{}
	HostUUIDCheck   bool
	EventsPoolSize  int
	CattleURL       string
	CattleAccessKey string
	CattleSecretKey string
	PidFile         string
	LogFile         string
}

var Config config

func ParsedPublicKey() error {
	keyBytes, err := ioutil.ReadFile(Config.Key)
	if err != nil {
		glog.Error("Error reading file")
		return err
	}

	block, _ := pem.Decode(keyBytes)
	pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return err
	}

	Config.ParsedPublicKey = pubKey

	return nil
}

func Parse() error {
	port, err := strconv.Atoi(utils.HostAPIPort())
	if err != nil {
		return err
	}
	Config.HaProxyMonitor = false
	Config.Port = port
	Config.IP = utils.HostAPIIP()
	Config.DockerURL = "unix:///var/run/docker.sock"
	Config.Auth = true
	if os.Getenv("CATTLE_PHYSICAL_HOST_UUID") == "" {
		Config.HostUUID = "DEFAULT"
	} else {
		Config.HostUUID = os.Getenv("CATTLE_PHYSICAL_HOST_UUID")
	}

	Config.HostUUIDCheck = true
	Config.Key = utils.JwtPublicKeyFile()
	Config.EventsPoolSize = 10
	Config.CattleURL = utils.APIURL("")
	Config.CattleAccessKey = utils.AccessKey()
	Config.CattleSecretKey = utils.SecretKey()

	if len(Config.Key) > 0 {
		if err := ParsedPublicKey(); err != nil {
			glog.Error("Error reading file")
			return err
		}
	}

	s, err := os.Stat("/run/systemd/system")
	if err != nil || !s.IsDir() {
		Config.Systemd = false
	} else {
		Config.Systemd = true
	}

	return nil
}
