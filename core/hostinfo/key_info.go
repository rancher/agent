package hostinfo

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
	"os"
	"path"

	"github.com/pkg/errors"
	"github.com/rancher/agent/utilities/config"
	"github.com/rancher/log"
)

type KeyCollector struct {
	key string
}

func (k KeyCollector) GetData() (map[string]interface{}, error) {
	key, err := k.getKey()
	return map[string]interface{}{
		"data": key,
	}, err
}

func (k KeyCollector) getKey() (string, error) {
	if k.key != "" {
		return k.key, nil
	}

	fileName := config.KeyFile()
	bytes, err := ioutil.ReadFile(fileName)
	if os.IsNotExist(err) {
		bytes, err = k.genKey()
		if err != nil {
			return "", err
		}
		os.MkdirAll(path.Dir(fileName), 0400)
		err = ioutil.WriteFile(fileName, bytes, 0400)
		if err != nil {
			return "", err
		}
	} else if err != nil {
		return "", err
	}

	b, _ := pem.Decode(bytes)
	key, err := x509.ParsePKCS1PrivateKey(b.Bytes)
	if err != nil {
		return "", err
	}

	bytes, err = x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal public key")
	}

	bytes = pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: bytes,
	})
	k.key = string(bytes)

	return k.key, nil
}

func (k KeyCollector) genKey() ([]byte, error) {
	log.Info("Generating host key")
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	log.Info("Done generating host key")
	return pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}), nil
}

func (k KeyCollector) KeyName() string {
	return "hostKey"
}

func (k KeyCollector) GetLabels(prefix string) (map[string]string, error) {
	return map[string]string{}, nil
}
