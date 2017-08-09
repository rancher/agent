// +build linux freebsd solaris openbsd darwin

package runtime

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/docker/go-connections/sockets"
	"github.com/pkg/errors"
	v3 "github.com/rancher/go-rancher/v3"
)

const (
	Create       = "Create"
	Remove       = "Remove"
	Attach       = "Attach"
	Mount        = "Mount"
	Path         = "Path"
	Unmount      = "Unmount"
	Get          = "Get"
	List         = "List"
	Capabilities = "Capabilities"
)

// Response is the strucutre that the plugin's responses are serialized to.
type Response struct {
	Mountpoint   string
	Err          string
	Volumes      []*Volume
	Volume       *Volume
	Capabilities Capability
}

// Volume represents a volume object for use with `Get` and `List` requests
type Volume struct {
	Name       string
	Mountpoint string
	Status     map[string]interface{}
}

// Capability represents the list of capabilities a volume driver can return
type Capability struct {
	Scope string
}

func callRancherStorageVolumeAttach(volume v3.Volume) error {
	transport := new(http.Transport)
	sockets.ConfigureTransport(transport, "unix", rancherStorageSockPath(volume))
	client := &http.Client{
		Transport: transport,
	}
	url := "http://volume-plugin/VolumeDriver.Attach"

	bs, err := json.Marshal(struct{ Name string }{Name: volume.Name})
	if err != nil {
		return errors.Wrap(err, "Failed to marshal JSON")
	}

	driver := volume.Driver
	resp, err := client.Post(url, "application/json", bytes.NewReader(bs))
	if err != nil {
		logrus.Errorf("Failed to call /VolumeDriver.Attach '%s' (driver '%s'): %s", volume.Name, driver, err)
		return err
	}
	data, err := ioutil.ReadAll(resp.Body)
	logrus.Info(string(data))
	defer resp.Body.Close()

	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		logrus.Infof("Success: /VolumeDriver.Attach '%s' (driver '%s')", volume.Name, driver)
	case resp.StatusCode >= 400 && resp.StatusCode < 500:
		logrus.Infof("/VolumeDriver.Attach '%s' is not supported by driver '%s'", volume.Name, driver)
	default:
		return errors.Errorf("/VolumeDriver.Attach '%s' (driver '%s') returned status %v: %s", volume.Name, driver, resp.StatusCode, resp.Status)
	}

	return nil
}

func callRancherStorageVolumePlugin(volume v3.Volume, action string, payload interface{}) (Response, error) {
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

	driver := volume.Driver
	resp, err := client.Post(url, "application/json", bytes.NewReader(bs))
	if err != nil {
		logrus.Errorf("Failed to call /VolumeDriver.%v '%s' (driver '%s'): %s", action, volume.Name, driver, err)
		return Response{}, err
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Response{}, err
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		logrus.Infof("Success: /VolumeDriver.%v '%s' (driver '%s')", action, volume.Name, driver)
		response := Response{}
		err := json.Unmarshal(data, &response)
		if err != nil {
			return Response{}, err
		}
		if response.Err != "" {
			return Response{}, errors.New(response.Err)
		}
		return response, nil
	case resp.StatusCode >= 400 && resp.StatusCode < 500:
		logrus.Infof("/VolumeDriver.%v '%s' is not supported by driver '%s'", action, volume.Name, driver)
	default:
		return Response{}, errors.Errorf("/VolumeDriver.Attach '%s' (driver '%s') returned status %v: %s", volume.Name, driver, resp.StatusCode, resp.Status)
	}

	return Response{}, nil
}
