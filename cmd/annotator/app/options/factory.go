package options

import (
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	coreinformers "k8s.io/client-go/informers/core/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// NewInformerFactory creates a SharedInformerFactory and initializes an event informer that returns specified events.
func NewInformerFactory(cs clientset.Interface, resyncPeriod time.Duration) informers.SharedInformerFactory {
	informerFactory := informers.NewSharedInformerFactory(cs, resyncPeriod)

	informerFactory.InformerFor(&v1.Event{}, newEventInformer)

	return informerFactory
}

// newEventInformer creates a shared index informer that returns only scheduled and normal event.
func newEventInformer(cs clientset.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	selector := fmt.Sprintf("type=%s,reason=Scheduled", v1.EventTypeNormal)

	tweakListOptions := func(options *metav1.ListOptions) {
		options.FieldSelector = selector
	}

	return coreinformers.NewFilteredEventInformer(cs, metav1.NamespaceAll, resyncPeriod, nil, tweakListOptions)
}
