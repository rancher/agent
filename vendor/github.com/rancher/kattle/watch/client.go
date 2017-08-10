package watch

import (
	"k8s.io/client-go/kubernetes"

	"k8s.io/client-go/pkg/api/v1"
)

type Client struct {
	clientset *kubernetes.Clientset
	pods      map[string]v1.Pod
	pvs       map[string]v1.PersistentVolume
	pvcs      map[string]v1.PersistentVolumeClaim
}

func NewClient(clientset *kubernetes.Clientset) *Client {
	return &Client{
		clientset,
		make(map[string]v1.Pod),
		make(map[string]v1.PersistentVolume),
		make(map[string]v1.PersistentVolumeClaim),
	}
}

func (c *Client) Start() {
	c.startPodWatch()
	c.startPvWatch()
	c.startPvcWatch()
}

func (c *Client) Pods() map[string]v1.Pod {
	return c.pods
}

func (c *Client) Pvs() map[string]v1.PersistentVolume {
	return c.pvs
}

func (c *Client) Pvcs() map[string]v1.PersistentVolumeClaim {
	return c.pvcs
}
