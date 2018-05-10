// +build linux freebsd solaris openbsd darwin

package storage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/docker/go-connections/sockets"
	"github.com/leodotcloud/log"
	"github.com/pkg/errors"
	"github.com/rancher/agent/model"
)

func CallRancherStorageVolumePlugin(volume model.Volume, action string, payload interface{}) (Response, error) {
	transport := new(http.Transport)
	sockets.ConfigureTransport(transport, "unix", rancherStorageSockPath(volume))
	client := &http.Client{
		Transport: transport,
	}
	url := fmt.Sprintf("http://volume-plugin/VolumeDriver.%v", action)

	bs, err := json.Marshal(payload)
	if err != nil {
		return Response{}, errors.Wrap(err, "Failed to marshal JSON")
	}

	driver := volume.Data.Fields.Driver
	resp, err := client.Post(url, "application/json", bytes.NewReader(bs))
	if err != nil {
		return Response{}, errors.Errorf("Failed to call /VolumeDriver.%v '%s' (driver '%s'): %s", action, volume.Name, driver, err)
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Response{}, err
	}
	defer resp.Body.Close()

	response := Response{}
	if err := json.Unmarshal(data, &response); err != nil {
		return Response{}, err
	}
	if response.Err != "" {
		return Response{}, errors.Errorf("Failed to %s volume %s. Driver: %s. Status code: %v. Status: %s. Error Message: %s", strings.ToLower(action), volume.Name, driver, resp.StatusCode, strings.ToLower(resp.Status), response.Err)
	}
	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		log.Infof("Success: /VolumeDriver.%v '%s' (driver '%s')", action, volume.Name, driver)
		return response, nil
	case resp.StatusCode >= 400 && resp.StatusCode < 500:
		log.Infof("/VolumeDriver.%v '%s' is not supported by driver '%s'", action, volume.Name, driver)
	default:
		return Response{}, errors.Errorf("Failed to %s volume %s. Driver: %s. Status code: %v. Status: %s", strings.ToLower(action), volume.Name, driver, resp.StatusCode, strings.ToLower(resp.Status))
	}

	return Response{}, nil
}
