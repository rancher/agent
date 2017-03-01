package filters

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"

	log "github.com/Sirupsen/logrus"
	"github.com/rancher/websocket-proxy/proxy/apiinterceptor/model"
)

type APIFilter interface {
	GetType() string
	ProcessFilter(filter model.FilterData, input model.APIRequestData) (model.APIRequestData, error)
}

func SignString(stringToSign []byte, sharedSecret []byte) string {
	h := hmac.New(sha512.New, sharedSecret)
	h.Write(stringToSign)

	signature := h.Sum(nil)
	encodedSignature := base64.URLEncoding.EncodeToString(signature)

	log.Debugf("Signature generated: %v", encodedSignature)

	return encodedSignature
}
