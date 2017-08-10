package sync

import "strings"

/*type VolumeSpec struct {
	Size string                  `json:"size,omitempty"`
	Raw  v1.PersistentVolumeSpec `json:",inline"`
}

func PvFromVolume(volume types.Volume) v1.PersistentVolume {
	volumeSpec := readVolumeSpec(volume)

	volumeName := getVolumeName(volume.Name)
	pv := v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: volumeName,
		},
	}

	pv.Spec = volumeSpec.Raw
	pv.Spec.StorageClassName = fmt.Sprintf("%s-storage-class", volumeName)
	//pv.Spec.AccessModes = volumeSpec.AccessModes

	if volumeSpec.Size != "" {
		// TODO: this can panic
		storageRequest := resource.MustParse(volumeSpec.Size)
		pv.Spec.Capacity = v1.ResourceList{
			"storage": storageRequest,
		}
	}

	return pv
}

func PvcFromVolume(volume types.Volume) v1.PersistentVolumeClaim {
	volumeSpec := readVolumeSpec(volume)

	volumeName := getVolumeName(volume.Name)
	storageClass := fmt.Sprintf("%s-storage-class", volumeName)
	claim := v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-claim", volumeName),
		},
		Spec: v1.PersistentVolumeClaimSpec{
			StorageClassName: &storageClass,
			AccessModes:      volumeSpec.Raw.AccessModes,
		},
	}

	if volumeSpec.Size != "" {
		// TODO: this can panic
		storageRequest := resource.MustParse(volumeSpec.Size)
		claim.Spec.Resources = v1.ResourceRequirements{
			Requests: v1.ResourceList{
				"storage": storageRequest,
			},
		}
	}

	return claim
}*/

func getVolumeName(volume string) string {
	return strings.Replace(volume, "_", "-", -1)
}

/*func readVolumeSpec(volume types.Volume) VolumeSpec {
	var raw v1.PersistentVolumeSpec
	if err := utils.ConvertByJSON(volume.Metadata, &raw); err != nil {
		// TODO
		panic(err)
	}

	var volumeSpec VolumeSpec
	volumeSpec.Raw = raw

	if err := utils.ConvertByJSON(volume.Metadata, &volumeSpec); err != nil {
		// TODO
		panic(err)
	}

	return volumeSpec
}*/
