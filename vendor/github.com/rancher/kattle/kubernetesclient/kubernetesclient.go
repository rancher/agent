package kubernetesclient

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func CreateKubernetesClient() (*kubernetes.Clientset, error) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(cfg)
}

func CreateRancherKubernetesClient() (*kubernetes.Clientset, error) {
	return nil, nil
}
