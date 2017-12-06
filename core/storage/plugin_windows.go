package storage

import (
	"github.com/rancher/agent/model"
)

// callRancherStorageVolumeAttach is not supported on windows
func callRancherStorageVolumeAttach(volume model.Volume) error {
	return nil
}
