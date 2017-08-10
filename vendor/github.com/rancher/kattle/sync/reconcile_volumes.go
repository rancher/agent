package sync

/*func reconcileVolumes(clientset *kubernetes.Clientset, watchClient *watch.Client, volumes []types.Volume, volumesIds map[string]bool) error {
	for _, volume := range volumes {
		if _, ok := volumesIds[volume.Id]; !ok {
			continue
		}

		go func(volume types.Volume) {
			// TODO: remove these hard-coded values
			volume.Metadata = map[string]interface{}{
				"accessModes": []string{
					"ReadWriteOnce",
				},
				"size": "8Gi",
				"nfs": map[string]interface{}{
					"server": "0.0.0.0",
					"path":   "/",
				},
			}

			pv := PvFromVolume(volume)
			if _, ok := watchClient.Pvs()[pv.Name]; !ok {
				if err := createPv(clientset, pv); err != nil {
					log.Error(err)
				}
			}

			pvc := PvcFromVolume(volume)
			if _, ok := watchClient.Pvcs()[pvc.Name]; !ok {
				if err := createPvc(clientset, pvc); err != nil {
					log.Error(err)
				}
			}
		}(volume)
	}
	return nil
}

func createPv(clientset *kubernetes.Clientset, pv v1.PersistentVolume) error {
	log.Infof("Creating PV %s", pv.Name)
	_, err := clientset.PersistentVolumes().Create(&pv)
	return err
}

func createPvc(clientset *kubernetes.Clientset, pvc v1.PersistentVolumeClaim) error {
	log.Infof("Creating PVC %s", pvc.Name)
	_, err := clientset.PersistentVolumeClaims(v1.NamespaceDefault).Create(&pvc)
	return err
}*/
