package compute

import (
	"fmt"
	"github.com/docker/engine-api/client"
	"github.com/pkg/errors"
	"github.com/rancher/agent/core/marshaller"
	"github.com/rancher/agent/model"
	"github.com/rancher/agent/utilities/config"
	"github.com/rancher/agent/utilities/constants"
	"github.com/rancher/agent/utilities/utils"
	"io/ioutil"
	"os"
	"path"
)

func RecordState(client *client.Client, instance model.Instance, dockerID string) error {
	if dockerID == "" {
		container, err := utils.GetContainer(client, instance, false)
		if err != nil && !utils.IsContainerNotFoundError(err) {
			return errors.Wrap(err, constants.RecordStateError+"failed to get container")
		}
		if container.ID != "" {
			dockerID = container.ID
		}
	}

	if dockerID == "" {
		return nil
	}
	contDir := config.ContainerStateDir()

	temFilePath := path.Join(contDir, fmt.Sprintf("tmp-%s", dockerID))
	if ok := utils.IsPathExist(temFilePath); ok {
		if err := os.Remove(temFilePath); err != nil {
			return errors.Wrap(err, constants.RecordStateError+"failed to remove file")
		}
	}

	filePath := path.Join(contDir, dockerID)
	if ok := utils.IsPathExist(temFilePath); ok {
		if err := os.Remove(filePath); err != nil {
			return errors.Wrap(err, constants.RecordStateError+"failed to remove file")
		}
	}

	if ok := utils.IsPathExist(contDir); !ok {
		mkErr := os.MkdirAll(contDir, 777)
		if mkErr != nil {
			return errors.Wrap(mkErr, constants.RecordStateError+"failed to make directory")
		}
	}

	data, err := marshaller.ToString(instance)
	if err != nil {
		return errors.Wrap(err, constants.RecordStateError+"failed to marshal data")
	}
	tempFile, err := ioutil.TempFile(contDir, "tmp-")

	if err != nil {
		return errors.Wrap(err, constants.RecordStateError+"failed to create tem files")
	}

	if writeErr := ioutil.WriteFile(tempFile.Name(), data, 0777); writeErr != nil {
		return errors.Wrap(writeErr, constants.RecordStateError+"failed to write data to tem file")
	}

	if err := tempFile.Close(); err != nil {
		return errors.Wrap(err, constants.RecordStateError+"failed to close file")
	}
	// this one is weird. Seems like the host-api is using the temp file and we can't rename the file
	// try it multiple times to wait for the host-api to release that file lock
	success := false
	for i := 0; i < 10; i++ {
		if err = os.Rename(tempFile.Name(), filePath); err == nil {
			success = true
			break
		}
	}
	if !success {
		return errors.Wrap(err, constants.RecordStateError+"failed to rename file")
	}

	return nil
}

func PurgeState(instance model.Instance, client *client.Client) error {
	container, err := utils.GetContainer(client, instance, false)
	if err != nil {
		if !utils.IsContainerNotFoundError(err) {
			return errors.Wrap(err, constants.PurgeStateError+"failed to get container")
		}
	}
	if container.ID == "" {
		return nil
	}
	dockerID := container.ID
	contDir := config.ContainerStateDir()
	files := []string{path.Join(contDir, "tmp-"+dockerID), path.Join(contDir, dockerID)}
	for _, f := range files {
		if _, err := os.Stat(f); err == nil {
			if rmErr := os.Remove(f); rmErr != nil {
				return errors.Wrap(rmErr, constants.PurgeStateError+"failed to remove file")
			}
		}
	}
	return nil
}
