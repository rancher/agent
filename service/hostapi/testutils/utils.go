package testutils

import (
	"io/ioutil"

	log "github.com/Sirupsen/logrus"
	jwt "github.com/dgrijalva/jwt-go"

	"github.com/rancher/websocket-proxy/proxy"
)

var privateKey interface{}

func ParseTestPrivateKey() interface{} {
	keyBytes, err := ioutil.ReadFile("../testutils/private.pem")
	if err != nil {
		log.Fatal("Failed to parse private key.", err)
	}

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(keyBytes)
	if err != nil {
		log.Fatal("Failed to parse private key.", err)
	}

	return privateKey
}

func GetTestConfig(addr string) *proxy.Config {
	config := &proxy.Config{
		ListenAddr: addr,
	}

	pubKey, err := proxy.ParsePublicKey("../testutils/public.pem")
	if err != nil {
		log.Fatal("Failed to parse key. ", err)
	}
	config.PublicKey = pubKey
	return config
}
