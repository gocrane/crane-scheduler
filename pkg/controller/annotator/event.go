package annotator

import (
	"fmt"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

type eventController struct {
	*Controller
	queue workqueue.RateLimitingInterface
}

func newEventController(c *Controller) *eventController {
	eventRateLimiter := workqueue.DefaultControllerRateLimiter()

	return &eventController{
		Controller: c,
		queue:      workqueue.NewNamedRateLimitingQueue(eventRateLimiter, "EVENT_event_queue"),
	}
}

func (e *eventController) Run() {
	defer e.queue.ShutDown()
	klog.Infof("Start to reconcile EVENT events")

	for e.processNextWorkItem() {
	}
}

func (e *eventController) processNextWorkItem() bool {
	key, quit := e.queue.Get()
	if quit {
		return false
	}
	defer e.queue.Done(key)

	err := e.reconcile(key.(string))
	if err != nil {
		klog.Warningf("failed to sync this EVENT [%q]: %v", key.(string), err)
	}

	return true
}

func (e *eventController) handles() cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc:    e.handleAddEvent,
		UpdateFunc: e.handleUpdateEvent,
	}
}

func (e *eventController) handleAddEvent(obj interface{}) {
	event := obj.(*v1.Event)

	if event.Type != v1.EventTypeNormal || event.Reason != "Scheduled" {
		return
	}

	e.enqueue(obj, cache.Added)
}

func (e *eventController) handleUpdateEvent(old, new interface{}) {
	oldEvent, curEvent := old.(*v1.Event), new.(*v1.Event)

	if oldEvent.ResourceVersion == curEvent.ResourceVersion {
		return
	}

	if curEvent.Type != v1.EventTypeNormal || curEvent.Reason != "Scheduled" {
		return
	}

	e.enqueue(new, cache.Updated)
}

func (e *eventController) enqueue(obj interface{}, action cache.DeltaType) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		return
	}

	klog.V(5).Infof("enqueue EVENT %s %s event", key, action)
	e.queue.Add(key)
}

func (e *eventController) reconcile(key string) error {
	startTime := time.Now()
	defer func() {
		klog.V(5).Infof("Finished syncing EVENT event %q (%v)", key, time.Since(startTime))
	}()

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	event, err := e.eventLister.Events(namespace).Get(name)
	if err != nil {
		return err
	}

	binding, err := translateEventToBinding(event)
	if err != nil {
		return err
	}

	e.bindingRecords.AddBinding(binding)

	return nil
}

func translateEventToBinding(event *v1.Event) (*Binding, error) {
	var metaKey, nodeName string

	_, err := fmt.Fscanf(strings.NewReader(event.Message), "Successfully assigned %s to %s", &metaKey, &nodeName)
	if err != nil {
		return nil, fmt.Errorf("failed to extact imformations from event message[%s]: %v", event.Message, err)
	}

	namespace, name, err := cache.SplitMetaNamespaceKey(metaKey)
	if err != nil {
		return nil, err
	}

	var lasteOccuredTime int64

	if event.Count == 0 {
		lasteOccuredTime = event.EventTime.Unix()
	} else {
		lasteOccuredTime = event.LastTimestamp.Unix()
	}

	return &Binding{
		Node:      nodeName,
		Namespace: namespace,
		PodName:   name,
		Timestamp: lasteOccuredTime,
	}, nil
}
