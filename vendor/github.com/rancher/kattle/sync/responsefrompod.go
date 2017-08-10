package sync

import (
	"github.com/rancher/go-rancher/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
)

const (
	rancherHostUuidLabel = "io.rancher.host.uuid"
)

func responseFromPod(pod v1.Pod) client.DeploymentSyncResponse {
	var instanceStatuses []client.InstanceStatus
	for _, containerStatus := range pod.Status.ContainerStatuses {
		// Don't report back on Rancher pause container
		if containerStatus.Name == rancherPauseContainerName {
			continue
		}

		state := ""
		// TODO: might be the wrong way to tell this
		if containerStatus.Ready {
			state = "running"
		}

		instanceStatuses = append(instanceStatuses, client.InstanceStatus{
			ExternalId:       containerStatus.ContainerID,
			InstanceUuid:     containerStatus.Name,
			PrimaryIpAddress: pod.Status.PodIP,
			State:            state,
		})
	}

	return client.DeploymentSyncResponse{
		InstanceStatus: instanceStatuses,
	}
}

func addHostUuidToResponse(clientset *kubernetes.Clientset, pod v1.Pod, response client.DeploymentSyncResponse) (client.DeploymentSyncResponse, error) {
	if pod.Spec.NodeName == "" {
		return response, nil
	}

	node, err := clientset.Nodes().Get(pod.Spec.NodeName, metav1.GetOptions{})
	if err != nil {
		return client.DeploymentSyncResponse{}, err
	}

	uuid, ok := node.Labels[rancherHostUuidLabel]
	if !ok {
		return response, nil
	}

	for i := range response.InstanceStatus {
		response.InstanceStatus[i].HostUuid = uuid
	}

	return response, nil
}
