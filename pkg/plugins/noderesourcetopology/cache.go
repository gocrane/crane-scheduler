package noderesourcetopology

import (
	"context"
	"fmt"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/scheduler/framework"

	topologyv1alpha1 "github.com/gocrane/api/topology/v1alpha1"
)

var (
	cleanAssumedPeriod = 1 * time.Second
)

// PodTopologyCache is a cache which stores the pod topology scheduling result.
// It is used before the pod bound since the result has not been recorded into
// annotations yet.
type PodTopologyCache interface {
	AssumePod(pod *corev1.Pod, zone topologyv1alpha1.ZoneList) error
	ForgetPod(pod *corev1.Pod) error
	PodCount() int
	GetPodTopology(pod *corev1.Pod) (topologyv1alpha1.ZoneList, error)
}

type podTopologyCacheImpl struct {
	// This mutex guards all fields within this cache struct.
	sync.RWMutex
	ttl            time.Duration
	period         time.Duration
	podTopology    map[string]topologyv1alpha1.ZoneList
	podTopologyTTL map[string]*time.Time
}

// NewPodTopologyCache returns a PodTopologyCache.
func NewPodTopologyCache(ctx context.Context, ttl time.Duration) PodTopologyCache {
	cache := &podTopologyCacheImpl{
		ttl:            ttl,
		period:         cleanAssumedPeriod,
		podTopology:    make(map[string]topologyv1alpha1.ZoneList),
		podTopologyTTL: make(map[string]*time.Time),
	}
	cache.run(ctx.Done())
	return cache
}

// AssumePod adds a pod and its topology result into cache.
func (c *podTopologyCacheImpl) AssumePod(pod *corev1.Pod, zone topologyv1alpha1.ZoneList) error {
	key, err := framework.GetPodKey(pod)
	if err != nil {
		return err
	}

	c.Lock()
	defer c.Unlock()

	if _, ok := c.podTopology[key]; ok {
		return fmt.Errorf("pod %v is in the podTopologyCache, so can't be assumed", key)
	}
	dl := time.Now().Add(c.ttl)
	c.podTopology[key] = zone
	c.podTopologyTTL[key] = &dl
	return nil
}

// ForgetPod removes the pod topology result from cache. It is called after binding or scheduling failure.
func (c *podTopologyCacheImpl) ForgetPod(pod *corev1.Pod) error {
	key, err := framework.GetPodKey(pod)
	if err != nil {
		return err
	}

	c.Lock()
	defer c.Unlock()

	c.removePod(key)
	return nil
}

// PodCount returns the number of pods in the cache.
func (c *podTopologyCacheImpl) PodCount() int {
	c.RLock()
	defer c.RUnlock()

	return len(c.podTopology)
}

// GetPodTopology returns the pod zone from the cache with the same namespace and the same name of the specified pod.
func (c *podTopologyCacheImpl) GetPodTopology(pod *corev1.Pod) (topologyv1alpha1.ZoneList, error) {
	key, err := framework.GetPodKey(pod)
	if err != nil {
		return nil, err
	}

	c.RLock()
	defer c.RUnlock()

	topology, ok := c.podTopology[key]
	if !ok {
		return nil, fmt.Errorf("pod topology %v does not exist in cache", key)
	}

	return topology, nil
}

func (c *podTopologyCacheImpl) run(stopCh <-chan struct{}) {
	go wait.Until(c.cleanupExpiredAssumedPods, c.period, stopCh)
}

func (c *podTopologyCacheImpl) cleanupExpiredAssumedPods() {
	c.cleanupAssumedPods(time.Now())
}

// cleanupAssumedPods exists for making test deterministic by taking time as input argument.
func (c *podTopologyCacheImpl) cleanupAssumedPods(now time.Time) {
	c.Lock()
	defer c.Unlock()

	for key, dl := range c.podTopologyTTL {
		if now.After(*dl) {
			c.removePod(key)
		}
	}
}

func (c *podTopologyCacheImpl) removePod(key string) {
	delete(c.podTopology, key)
	delete(c.podTopologyTTL, key)
	klog.V(4).Infof("Finished binding for pod %v. Can be expired.", key)
}
