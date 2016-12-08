package proxy

import (
	"archive/zip"
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/rancher/go-rancher/client"

	"github.com/Sirupsen/logrus"
	"github.com/rakyll/globalconf"
)

type Config struct {
	PublicKey            interface{}
	ListenAddr           string
	CattleAddr           string
	ParentPid            int
	ProxyProtoHTTPSPorts map[int]bool
	CattleAccessKey      string
	CattleSecretKey      string
	TLSListenAddr        string
}

func GetConfig() (*Config, error) {
	c := &Config{
		CattleAccessKey: os.Getenv("CATTLE_ACCESS_KEY"),
		CattleSecretKey: os.Getenv("CATTLE_SECRET_KEY"),
	}
	var keyFile string
	var keyContents string
	var proxyProtoHTTPSPorts string
	flag.StringVar(&keyFile, "jwt-public-key-file", "", "Location of the public-key used to validate JWTs.")
	flag.StringVar(&keyContents, "jwt-public-key-contents", "", "An alternative to jwt-public-key-file. The contents of the key.")
	flag.StringVar(&c.ListenAddr, "listen-address", ":8080", "The tcp address to listen on.")
	flag.StringVar(&c.TLSListenAddr, "tls-listen-address", "", "The tcp address to listen on for swarm.")
	flag.StringVar(&c.CattleAddr, "cattle-address", "", "The tcp address to forward cattle API requests to. Will not proxy to cattle api if this option is not provied.")
	flag.IntVar(&c.ParentPid, "parent-pid", 0, "If provided, this process will exit when the specified parent process stops running.")
	flag.StringVar(&proxyProtoHTTPSPorts, "https-proxy-protocol-ports", "", "If proxy protocol is used, a list of proxy ports that will allow us to recognize that the connection was over https.")

	confOptions := &globalconf.Options{
		EnvPrefix: "PROXY_",
	}

	conf, err := globalconf.NewWithOptions(confOptions)

	if err != nil {
		return nil, err
	}

	conf.ParseAll()

	if keyFile != "" && keyContents != "" {
		return nil, fmt.Errorf("Can't specify both jwt-public-key-file and jwt-public-key-contents")
	}
	var parsedKey interface{}
	var parseErr error
	if keyFile != "" {
		parsedKey, parseErr = ParsePublicKey(keyFile)
	} else if keyContents != "" {
		parsedKey, parseErr = ParsePublicKeyFromMemory(keyContents)
	} else if c.CattleAddr != "" {
		bytes, err := downloadKey(c.CattleAddr)
		if err != nil {
			parseErr = err
		}
		parsedKey, parseErr = publicKeyDecode(bytes)
	} else {
		parseErr = fmt.Errorf("Must specify one of jwt-public-key-file and jwt-public-key-contents")
	}
	if parseErr != nil {
		return nil, parseErr
	}

	c.PublicKey = parsedKey

	portMap := make(map[int]bool)
	ports := strings.Split(proxyProtoHTTPSPorts, ",")
	for _, port := range ports {
		if p, err := strconv.Atoi(port); err == nil {
			portMap[p] = true
		}
	}
	c.ProxyProtoHTTPSPorts = portMap

	return c, nil
}

func (config *Config) GetCerts() (*Certs, error) {
	return downloadCert(config.CattleAccessKey, config.CattleSecretKey, config.CattleAddr)
}

func ParsePublicKey(keyFile string) (interface{}, error) {
	keyBytes, err := ioutil.ReadFile(keyFile)
	if err != nil {
		return nil, err
	}

	return publicKeyDecode(keyBytes)
}

func ParsePublicKeyFromMemory(keyFileContents string) (interface{}, error) {
	return publicKeyDecode([]byte(keyFileContents))
}

func publicKeyDecode(keyBytes []byte) (interface{}, error) {
	block, _ := pem.Decode(keyBytes)
	if block == nil {
		return nil, errors.New("Invalid key content")
	}
	pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	return pubKey, nil
}

func downloadKey(addr string) ([]byte, error) {
	url := fmt.Sprintf("http://%s/v1/scripts/api.crt", addr)
	logrus.Infof("Downloading key from %s", url)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	buffer := &bytes.Buffer{}
	_, err = io.Copy(buffer, resp.Body)
	return buffer.Bytes(), err
}

type Certs struct {
	CA   []byte
	Cert []byte
	Key  []byte
}

func downloadCert(accessKey, secretKey, addr string) (*Certs, error) {
	url := fmt.Sprintf("http://%s/v1/schemas", addr)
	c, err := client.NewRancherClient(&client.ClientOpts{
		Url:       url,
		AccessKey: accessKey,
		SecretKey: secretKey,
	})
	if err != nil {
		return nil, err
	}

	opts := client.NewListOpts()
	opts.Filters["publicValue"] = accessKey
	opts.Filters["kind"] = "agentApiKey"
	certs, err := c.Credential.List(opts)
	if err != nil || len(certs.Data) == 0 {
		return nil, fmt.Errorf("Failed to download certificate for %s: %v", accessKey, err)
	}

	downloadURL := certs.Data[0].Links["certificate"]
	logrus.Infof("Downloading certificate from %s", downloadURL)
	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(accessKey, secretKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	buffer := &bytes.Buffer{}
	if _, err := io.Copy(buffer, resp.Body); err != nil {
		return nil, err
	}

	zipReader, err := zip.NewReader(bytes.NewReader(buffer.Bytes()), int64(buffer.Len()))
	if err != nil {
		return nil, err
	}

	result := &Certs{}

	for _, f := range zipReader.File {
		in, err := f.Open()
		if err != nil {
			return nil, err
		}

		buffer := &bytes.Buffer{}
		if _, err := io.Copy(buffer, in); err != nil {
			return nil, err
		}

		switch f.Name {
		case "ca.pem":
			result.CA = buffer.Bytes()
		case "cert.pem":
			result.Cert = buffer.Bytes()
		case "key.pem":
			result.Key = buffer.Bytes()
		}

		in.Close()
	}

	return result, nil
}
