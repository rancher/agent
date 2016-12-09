package auth

import (
	"net/http"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/golang/glog"
	"github.com/rancher/agent/service/hostapi/app/common"
	"github.com/rancher/agent/service/hostapi/config"
)

func Auth(rw http.ResponseWriter, req *http.Request) bool {
	if !config.Config.Auth {
		return true
	}
	tokenString := req.URL.Query().Get("token")

	if len(tokenString) == 0 {
		return false
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return config.Config.ParsedPublicKey, nil
	})
	SetToken(req, token)

	if err != nil {
		common.CheckError(err, 2)
		return false
	}

	if !token.Valid {
		return false
	}

	if config.Config.HostUUIDCheck && token.Claims["hostUuid"] != config.Config.HostUUID {
		glog.Infoln("Host UUID mismatch , authentication failed")
		return false
	}

	return true
}

func GetAndCheckToken(tokenString string) (*jwt.Token, bool) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return config.Config.ParsedPublicKey, nil
	})
	if err != nil {
		common.CheckError(err, 2)
		return token, false
	}

	if !token.Valid {
		return token, false
	}

	if config.Config.HostUUIDCheck && token.Claims["hostUuid"] != config.Config.HostUUID {
		glog.Infoln("Host UUID mismatch , authentication failed")
		return token, false
	}

	return token, true

}
