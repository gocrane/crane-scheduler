package noderesourcetopology

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"

	topologyv1alpha1 "github.com/gocrane/api/topology/v1alpha1"

	"github.com/gocrane/crane-scheduler/pkg/utils"
)

const (
	ErrReasonNUMAResourceNotEnough = "node(s) had insufficient resource of NUMA node"
	ErrReasonFailedToGetNRT        = "node(s) failed to get NRT"
)

// PreFilter invoked at the prefilter extension point.
func (tm *TopologyMatch) PreFilter(
	ctx context.Context,
	state *framework.CycleState,
	pod *corev1.Pod,
) (*framework.PreFilterResult, *framework.Status) {
	var indices []int
	if tm.topologyAwareResources.Has(string(corev1.ResourceCPU)) {
		indices = GetPodTargetContainerIndices(pod)
	}
	resources := computeContainerSpecifiedResourceRequest(pod, indices, tm.topologyAwareResources)
	state.Write(stateKey, &stateData{
		aware:                   IsPodAwareOfTopology(pod.Annotations),
		targetContainerIndices:  indices,
		targetContainerResource: resources,
		podTopologyByNode:       make(map[string]*nodeWrapper),
	})
	return nil, nil
}

// PreFilterExtensions returns prefilter extensions, pod add and remove.
func (tm *TopologyMatch) PreFilterExtensions() framework.PreFilterExtensions {
	return nil
}

// Filter will check if there exists numa node which has sufficient resource for the pod.
func (tm *TopologyMatch) Filter(
	ctx context.Context,
	state *framework.CycleState,
	pod *corev1.Pod,
	nodeInfo *framework.NodeInfo,
) *framework.Status {
	s, err := getStateData(state)
	if err != nil {
		return framework.AsStatus(err)
	}

	if nodeInfo.Node() == nil {
		return framework.NewStatus(framework.Error, "node(s) not found")
	}

	if utils.IsDaemonsetPod(pod) || len(s.targetContainerIndices) == 0 {
		return nil
	}

	nrt, err := tm.lister.Get(nodeInfo.Node().Name)
	if err != nil {
		return framework.NewStatus(framework.Unschedulable, ErrReasonFailedToGetNRT)
	}
	// let kubelet handle cpuset
	if nrt.CraneManagerPolicy.CPUManagerPolicy != topologyv1alpha1.CPUManagerPolicyStatic {
		return nil
	}

	nw := tm.initializeNodeWrapper(s, nodeInfo, nrt)
	if nw.aware {
		if status := tm.filterNUMANodeResource(s, nw); status != nil {
			return status
		}
	}
	assignTopologyResult(nw, s.targetContainerResource.Clone())

	s.Lock()
	defer s.Unlock()
	s.podTopologyByNode[nw.node] = nw

	return nil
}

func (tm *TopologyMatch) initializeNodeWrapper(
	state *stateData,
	nodeInfo *framework.NodeInfo,
	nrt *topologyv1alpha1.NodeResourceTopology,
) *nodeWrapper {
	node := nodeInfo.Node()
	nw := newNodeWrapper(node.Name, tm.topologyAwareResources, nrt.Zones, tm.GetPodTopology)
	for _, pod := range nodeInfo.Pods {
		nw.addPod(pod.Pod)
	}
	// If pod has specified awareness, ignore the awareness of node.
	if state.aware != nil {
		nw.aware = *state.aware
	} else {
		nw.aware = isNodeAwareOfTopology(nrt)
	}
	return nw
}

func (tm *TopologyMatch) filterNUMANodeResource(state *stateData, nw *nodeWrapper) *framework.Status {
	var res []*numaNode
	for _, numaNode := range nw.numaNodes {
		// Check resource
		insufficientResources := fitsRequestForNUMANode(state.targetContainerResource, numaNode)
		if len(insufficientResources) != 0 {
			continue
		}
		res = append(res, numaNode)
	}

	if len(res) == 0 {
		return framework.NewStatus(framework.Unschedulable, ErrReasonNUMAResourceNotEnough)
	}
	nw.numaNodes = res
	return nil
}

func isNodeAwareOfTopology(nrt *topologyv1alpha1.NodeResourceTopology) bool {
	return nrt.CraneManagerPolicy.TopologyManagerPolicy == topologyv1alpha1.TopologyManagerPolicySingleNUMANodePodLevel
}
