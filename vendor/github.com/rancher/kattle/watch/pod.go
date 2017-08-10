package watch

import (
	"time"

	"github.com/rancher/kattle/labels"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
)

func (c *Client) startPodWatch() chan struct{} {
	watchlist := cache.NewListWatchFromClient(c.clientset.Core().RESTClient(), "pods", v1.NamespaceDefault, fields.Everything())
	_, controller := cache.NewInformer(
		watchlist,
		&v1.Pod{},
		time.Second*0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: podFilterAddDelete(func(pod v1.Pod) {
				c.pods[pod.Name] = pod
			}),
			DeleteFunc: podFilterAddDelete(func(pod v1.Pod) {
				delete(c.pods, pod.Name)
			}),
			UpdateFunc: podFilterUpdate(func(pod v1.Pod) {
				c.pods[pod.Name] = pod
			}),
		},
	)

	stop := make(chan struct{})
	go controller.Run(stop)

	return stop
}

func podFilterAddDelete(f func(v1.Pod)) func(interface{}) {
	return func(obj interface{}) {
		pod := obj.(*v1.Pod)
		if _, ok := pod.Labels[labels.RevisionLabel]; ok {
			f(*pod)
		}
	}
}

func podFilterUpdate(f func(v1.Pod)) func(interface{}, interface{}) {
	return func(oldObj, newObj interface{}) {
		podFilterAddDelete(f)(newObj)
	}
}
