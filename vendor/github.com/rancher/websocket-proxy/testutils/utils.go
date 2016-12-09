package testutils

import (
	"io/ioutil"

	log "github.com/Sirupsen/logrus"
	jwt "github.com/dgrijalva/jwt-go"
)

func CreateTokenWithPayload(payload map[string]interface{}, privateKey interface{}) string {
	token := jwt.New(jwt.GetSigningMethod("RS256"))
	token.Claims = payload
	signed, err := token.SignedString(privateKey)
	if err != nil {
		log.Fatal("Failed to parse private key.", err)
	}
	return signed
}

func CreateToken(hostUUID string, privateKey interface{}) string {
	token := jwt.New(jwt.GetSigningMethod("RS256"))
	token.Claims["hostUuid"] = hostUUID
	signed, err := token.SignedString(privateKey)
	if err != nil {
		log.Fatal("Failed to parse private key.", err)
	}
	return signed
}

func CreateBackendToken(reportedUUID string, privateKey interface{}) string {
	token := jwt.New(jwt.GetSigningMethod("RS256"))
	token.Claims["reportedUuid"] = reportedUUID
	signed, err := token.SignedString(privateKey)
	if err != nil {
		log.Fatal("Failed to parse private key.", err)
	}
	return signed
}

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

func ParseTestPublicKey() interface{} {
	keyBytes, err := ioutil.ReadFile("../testutils/public.pem")
	if err != nil {
		log.Fatal("Failed to parse public key.", err)
	}

	publicKey, err := jwt.ParseRSAPublicKeyFromPEM(keyBytes)
	if err != nil {
		log.Fatal("Failed to parse public key.", err)
	}

	return publicKey

}
