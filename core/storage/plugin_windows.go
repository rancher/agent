package storage

// callRancherStorageVolumeAttach is not supported on windows
func callRancherStorageVolumeAttach(volume model.Volume) error {
	return nil
}
