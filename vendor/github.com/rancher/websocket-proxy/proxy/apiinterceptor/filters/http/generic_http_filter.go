package http

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/rancher/websocket-proxy/proxy/apiinterceptor/filters"
	"github.com/rancher/websocket-proxy/proxy/apiinterceptor/model"
)

const (
	interceptorType = "http"
)

type GenericHTTPFilter struct {
	client *http.Client
}

func (f *GenericHTTPFilter) GetType() string {
	return interceptorType
}

func NewFilter() (filters.APIFilter, error) {
	httpFilter := &GenericHTTPFilter{
		client: &http.Client{},
	}
	log.Infof("Configured %s API filter", httpFilter.GetType())

	return httpFilter, nil
}

func (f *GenericHTTPFilter) ProcessFilter(filter model.FilterData, input model.APIRequestData) (model.APIRequestData, error) {
	output := model.APIRequestData{}
	bodyContent, err := json.Marshal(input)
	if err != nil {
		return output, err
	}

	log.Debugf("Request => %s", bodyContent)

	req, err := http.NewRequest("POST", filter.Endpoint, bytes.NewBuffer(bodyContent))
	if err != nil {
		return output, err
	}
	//sign the body
	if filter.SecretToken != "" {
		signature := filters.SignString(bodyContent, []byte(filter.SecretToken))
		req.Header.Set(model.SignatureHeader, signature)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Length", string(len(bodyContent)))

	var tout int
	if filter.Timeout == "0" || filter.Timeout == "" {
		tout = 15
	} else {
		var err error
		tout, err = strconv.Atoi(filter.Timeout)
		if err != nil {
			tout = 15
		}
	}
	f.client.Timeout = time.Second * time.Duration(tout)
	resp, err := f.client.Do(req)
	if err != nil {
		return output, err
	}
	log.Debugf("Response Status <= " + resp.Status)
	defer resp.Body.Close()

	byteContent, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return output, err
	}

	log.Debugf("Response <= %s", byteContent)

	json.Unmarshal(byteContent, &output)
	output.Status = resp.StatusCode

	return output, nil
}
