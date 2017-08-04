package runtime

import "github.com/rancher/agent/model"

// callRancherStorageVolumeAttach is not supported on windows
func callRancherStorageVolumeAttach(volume types.Volume) error {
	return nil
}
