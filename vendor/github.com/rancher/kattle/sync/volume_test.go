package sync

/*var (
	testVolume = types.Volume{
		Volume: client.Volume{
			Name: "v",
		},
		Metadata: map[string]interface{}{
			"accessModes": []string{
				"ReadWriteOnce",
			},
			"size": "8Gi",
			"nfs": map[string]interface{}{
				"path":   "/tmp",
				"server": "172.17.0.2",
			},
		},
	}
)

func TestPvFromVolume(t *testing.T) {
	assert.Equal(t, PvFromVolume(testVolume), v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: "v",
		},
		Spec: v1.PersistentVolumeSpec{
			StorageClassName: "v-storage-class",
			AccessModes: []v1.PersistentVolumeAccessMode{
				v1.PersistentVolumeAccessMode("ReadWriteOnce"),
			},
			Capacity: v1.ResourceList{
				"storage": resource.MustParse("8Gi"),
			},
			PersistentVolumeSource: v1.PersistentVolumeSource{
				NFS: &v1.NFSVolumeSource{
					Path:   "/tmp",
					Server: "172.17.0.2",
				},
			},
		},
	})
}

func TestPvcFromVolume(t *testing.T) {
	assert.Equal(t, PvcFromVolume(testVolume), v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: "v-claim",
		},
		Spec: v1.PersistentVolumeClaimSpec{
			StorageClassName: &[]string{"v-storage-class"}[0],
			AccessModes: []v1.PersistentVolumeAccessMode{
				v1.PersistentVolumeAccessMode("ReadWriteOnce"),
			},
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					"storage": resource.MustParse("8Gi"),
				},
			},
		},
	})
}*/
