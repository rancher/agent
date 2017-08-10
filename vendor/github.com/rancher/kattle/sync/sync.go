package sync

import (
	log "github.com/Sirupsen/logrus"
	"github.com/rancher/go-rancher/v3"
	"github.com/rancher/kattle/watch"
	"k8s.io/client-go/kubernetes"
)

func Activate(clientset *kubernetes.Clientset, watchClient *watch.Client, deploymentUnit client.DeploymentSyncRequest) (client.DeploymentSyncResponse, error) {
	/*volumeIds := map[string]bool{}
	for _, deploymentUnit := range deploymentUnits {
		for _, container := range deploymentUnit.Containers {
			for _, mount := range container.Mounts {
				volumeIds[mount.VolumeId] = true
			}
		}
	}

	if err := reconcileVolumes(clientset, watchClient, volumes, volumeIds); err != nil {
		return err
	}*/

	pod := PodFromDeploymentUnit(deploymentUnit)
	createdPod, err := reconcilePod(clientset, watchClient, pod)
	if err != nil {
		return client.DeploymentSyncResponse{}, err
	}

	response := responseFromPod(createdPod)
	return addHostUuidToResponse(clientset, createdPod, response)
}

// TODO
func Remove(clientset *kubernetes.Clientset, watchClient *watch.Client, deploymentUnit client.DeploymentSyncRequest) error {
	log.Info("Remove")
	return nil
}
