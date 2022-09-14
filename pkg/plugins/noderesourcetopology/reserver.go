package noderesourcetopology

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

// Reserve reserves topology of pod and saves status in cycle stateData.
func (tm *TopologyMatch) Reserve(
	ctx context.Context,
	state *framework.CycleState,
	pod *corev1.Pod,
	nodeName string,
) *framework.Status {
	s, err := getStateData(state)
	if err != nil {
		return framework.AsStatus(err)
	}
	nw, exist := s.podTopologyByNode[nodeName]
	if !exist {
		return nil
	}
	if len(nw.result) == 0 {
		// Should never happen
		return framework.NewStatus(framework.Error, "node(s) topology result is empty")
	}
	s.topologyResult = nw.result
	// Assume pod
	if err = tm.AssumePod(pod, s.topologyResult); err != nil {
		return framework.AsStatus(err)
	}
	return nil
}

// Unreserve clears assumed Pod topology cache.
// It's idempotent, and does nothing if no cache found for the given pod.
func (tm *TopologyMatch) Unreserve(ctx context.Context, state *framework.CycleState, pod *corev1.Pod, nodeName string) {
	s, err := getStateData(state)
	if err != nil {
		return
	}
	_, exist := s.podTopologyByNode[nodeName]
	if !exist {
		return
	}
	if err = tm.ForgetPod(pod); err != nil {
		return
	}
}
