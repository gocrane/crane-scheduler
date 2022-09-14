package noderesourcetopology

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/sets"
	v1helper "k8s.io/kubernetes/pkg/apis/core/v1/helper"
	v1qos "k8s.io/kubernetes/pkg/apis/core/v1/helper/qos"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/noderesources"

	topologyv1alpha1 "github.com/gocrane/api/topology/v1alpha1"
)

var (
	// SupportedPolicy is the valid cpu policy.
	SupportedPolicy = sets.NewString(
		topologyv1alpha1.AnnotationPodCPUPolicyNone, topologyv1alpha1.AnnotationPodCPUPolicyExclusive,
		topologyv1alpha1.AnnotationPodCPUPolicyNUMA, topologyv1alpha1.AnnotationPodCPUPolicyImmovable,
	)
)

// IsPodAwareOfTopology returns if the pod needs to be scheduled based on topology awareness.
func IsPodAwareOfTopology(attr map[string]string) *bool {
	if val, exist := attr[topologyv1alpha1.AnnotationPodTopologyAwarenessKey]; exist {
		if awareness, err := strconv.ParseBool(val); err == nil {
			return &awareness
		}
	}
	return nil
}

// GetPodTargetContainerIndices returns all pod whose cpus could be bound.
// The pod is running, so we ignore all init containers.
func GetPodTargetContainerIndices(pod *corev1.Pod) []int {
	if policy := GetPodCPUPolicy(pod.Annotations); policy == topologyv1alpha1.AnnotationPodCPUPolicyNone {
		return nil
	}
	if v1qos.GetPodQOS(pod) != corev1.PodQOSGuaranteed {
		return nil
	}
	var idx []int
	for i := range pod.Spec.Containers {
		if GuaranteedCPUs(&pod.Spec.Containers[i]) > 0 {
			idx = append(idx, i)
		}
	}
	return idx
}

// GetPodCPUPolicy returns the cpu policy of pod, only supports none, exclusive and numa.
func GetPodCPUPolicy(attr map[string]string) string {
	policy, ok := attr[topologyv1alpha1.AnnotationPodCPUPolicyKey]
	if ok && SupportedPolicy.Has(policy) {
		return policy
	}
	return ""
}

// GuaranteedCPUs returns CPUs for guaranteed pod.
func GuaranteedCPUs(container *corev1.Container) int {
	cpuQuantity := container.Resources.Requests[corev1.ResourceCPU]
	if cpuQuantity.Value()*1000 != cpuQuantity.MilliValue() {
		return 0
	}
	// Safe downcast to do for all systems with < 2.1 billion CPUs.
	// Per the language spec, `int` is guaranteed to be at least 32 bits wide.
	// https://golang.org/ref/spec#Numeric_types
	return int(cpuQuantity.Value())
}

// GetPodTopologyResult returns the Topology scheduling result of a pod.
func GetPodTopologyResult(pod *corev1.Pod) topologyv1alpha1.ZoneList {
	raw, exist := pod.Annotations[topologyv1alpha1.AnnotationPodTopologyResultKey]
	if !exist {
		return nil
	}
	var zones topologyv1alpha1.ZoneList
	if err := json.Unmarshal([]byte(raw), &zones); err != nil {
		return nil
	}
	return zones
}

// GetPodNUMANodeResult returns the NUMA node scheduling result of a pod.
func GetPodNUMANodeResult(pod *corev1.Pod) topologyv1alpha1.ZoneList {
	zones := GetPodTopologyResult(pod)
	var numaZones topologyv1alpha1.ZoneList
	for i := range zones {
		if zones[i].Type == topologyv1alpha1.ZoneTypeNode {
			numaZones = append(numaZones, zones[i])
		}
	}
	return numaZones
}

type getAssumedPodTopologyFunc func(pod *corev1.Pod) (topologyv1alpha1.ZoneList, error)

type numaNode struct {
	name        string
	allocatable *framework.Resource
	requested   *framework.Resource
}

func newNumaNode(zone *topologyv1alpha1.Zone) *numaNode {
	var allocatable corev1.ResourceList
	if zone.Resources != nil {
		allocatable = zone.Resources.Allocatable
	}
	return &numaNode{
		name:        zone.Name,
		allocatable: framework.NewResource(allocatable),
		requested:   &framework.Resource{},
	}
}

func (nn *numaNode) addResource(info *topologyv1alpha1.ResourceInfo) {
	if info == nil {
		return
	}
	nn.requested.Add(info.Capacity)
}

type nodeWrapper struct {
	aware                 bool
	node                  string
	numaNodes             []*numaNode
	getAssumedPodTopology getAssumedPodTopologyFunc
	// we only care about the specified resources.
	topologyAwareResources sets.String
	result                 topologyv1alpha1.ZoneList
}

func newNodeWrapper(
	node string,
	resourceNames sets.String,
	zones topologyv1alpha1.ZoneList,
	f getAssumedPodTopologyFunc,
) *nodeWrapper {
	nw := &nodeWrapper{node: node, getAssumedPodTopology: f, topologyAwareResources: resourceNames}
	for i := range zones {
		nw.numaNodes = append(nw.numaNodes, newNumaNode(&zones[i]))
	}
	return nw
}

func (nw *nodeWrapper) addPod(pod *corev1.Pod) {
	numaNodeResult := GetPodNUMANodeResult(pod)
	// If result not found, we check the assumed cache because pod may not be bound.
	if len(numaNodeResult) == 0 {
		var err error
		if numaNodeResult, err = nw.getAssumedPodTopology(pod); err != nil {
			return
		}
	}
	nw.addNUMAResources(numaNodeResult)
}

func (nw *nodeWrapper) addNUMAResources(numaNodeResult topologyv1alpha1.ZoneList) {
	for i := range numaNodeResult {
		result := &numaNodeResult[i]
		for _, node := range nw.numaNodes {
			if node.name == result.Name {
				node.addResource(result.Resources)
			}
		}
	}
}

func assignTopologyResult(nw *nodeWrapper, request *framework.Resource) {
	// sort by free CPU resource
	sort.Slice(nw.numaNodes, func(i, j int) bool {
		nodeI, nodeJ := nw.numaNodes[i], nw.numaNodes[j]
		return nodeI.allocatable.MilliCPU-nodeI.requested.MilliCPU > nodeJ.allocatable.MilliCPU-nodeJ.requested.MilliCPU
	})

	if nw.aware {
		nw.result = []topologyv1alpha1.Zone{
			{
				Name: nw.numaNodes[0].name,
				Type: topologyv1alpha1.ZoneTypeNode,
				Resources: &topologyv1alpha1.ResourceInfo{
					Capacity: ResourceListIgnoreZeroResources(request),
				},
			},
		}
		return
	}

	for _, node := range nw.numaNodes {
		node.allocatable.MilliCPU = node.allocatable.MilliCPU / 1000 * 1000
		res, finished := assignRequestForNUMANode(request, node)
		if capacity := ResourceListIgnoreZeroResources(res); len(capacity) != 0 {
			nw.result = append(nw.result, topologyv1alpha1.Zone{
				Name: node.name,
				Type: topologyv1alpha1.ZoneTypeNode,
				Resources: &topologyv1alpha1.ResourceInfo{
					Capacity: ResourceListIgnoreZeroResources(res),
				},
			})
		}
		if finished {
			break
		}
	}
	sort.Slice(nw.result, func(i, j int) bool {
		return nw.result[i].Name < nw.result[j].Name
	})
}

func computeContainerSpecifiedResourceRequest(pod *corev1.Pod, indices []int, names sets.String) *framework.Resource {
	result := &framework.Resource{}
	for _, idx := range indices {
		container := &pod.Spec.Containers[idx]
		resources := make(corev1.ResourceList)
		for resourceName := range container.Resources.Requests {
			if names.Has(string(resourceName)) {
				resources[resourceName] = container.Resources.Requests[resourceName]
			}
		}
		result.Add(resources)
	}

	return result
}

func fitsRequestForNUMANode(podRequest *framework.Resource, numaNode *numaNode) []noderesources.InsufficientResource {
	insufficientResources := make([]noderesources.InsufficientResource, 0, 3)
	allocatable := numaNode.allocatable
	requested := numaNode.requested
	if podRequest.MilliCPU == 0 &&
		podRequest.Memory == 0 &&
		podRequest.EphemeralStorage == 0 &&
		len(podRequest.ScalarResources) == 0 {
		return insufficientResources
	}

	if podRequest.MilliCPU > (allocatable.MilliCPU - requested.MilliCPU) {
		insufficientResources = append(insufficientResources, noderesources.InsufficientResource{
			ResourceName: corev1.ResourceCPU,
			Reason:       "Insufficient cpu of NUMA node",
			Requested:    podRequest.MilliCPU,
			Used:         requested.MilliCPU,
			Capacity:     allocatable.MilliCPU,
		})
	}
	if podRequest.Memory > (allocatable.Memory - requested.Memory) {
		insufficientResources = append(insufficientResources, noderesources.InsufficientResource{
			ResourceName: corev1.ResourceMemory,
			Reason:       "Insufficient memory of NUMA node",
			Requested:    podRequest.Memory,
			Used:         requested.Memory,
			Capacity:     allocatable.Memory,
		})
	}
	if podRequest.EphemeralStorage > (allocatable.EphemeralStorage - requested.EphemeralStorage) {
		insufficientResources = append(insufficientResources, noderesources.InsufficientResource{
			ResourceName: corev1.ResourceEphemeralStorage,
			Reason:       "Insufficient ephemeral-storage of NUMA node",
			Requested:    podRequest.EphemeralStorage,
			Used:         requested.EphemeralStorage,
			Capacity:     allocatable.EphemeralStorage,
		})
	}

	for rName, rQuant := range podRequest.ScalarResources {
		if rQuant > (allocatable.ScalarResources[rName] - requested.ScalarResources[rName]) {
			insufficientResources = append(insufficientResources, noderesources.InsufficientResource{
				ResourceName: rName,
				Reason:       fmt.Sprintf("Insufficient %v of NUMA node", rName),
				Requested:    podRequest.ScalarResources[rName],
				Used:         requested.ScalarResources[rName],
				Capacity:     allocatable.ScalarResources[rName],
			})
		}
	}

	return insufficientResources
}

func assignRequestForNUMANode(podRequest *framework.Resource, numaNode *numaNode) (*framework.Resource, bool) {
	allocatable := numaNode.allocatable
	requested := numaNode.requested
	if podRequest.MilliCPU == 0 &&
		podRequest.Memory == 0 &&
		podRequest.EphemeralStorage == 0 &&
		len(podRequest.ScalarResources) == 0 {
		return nil, false
	}

	res := &framework.Resource{}
	finished := true

	assigned := min(podRequest.MilliCPU, allocatable.MilliCPU-requested.MilliCPU)
	podRequest.MilliCPU -= assigned
	res.MilliCPU = assigned
	if podRequest.MilliCPU > 0 {
		finished = false
	}

	assigned = min(podRequest.Memory, allocatable.Memory-requested.Memory)
	podRequest.Memory -= assigned
	res.Memory = assigned
	if podRequest.Memory > 0 {
		finished = false
	}

	assigned = min(podRequest.EphemeralStorage, allocatable.EphemeralStorage-requested.EphemeralStorage)
	podRequest.EphemeralStorage -= assigned
	res.EphemeralStorage = assigned
	if podRequest.EphemeralStorage > 0 {
		finished = false
	}

	for rName, rQuant := range podRequest.ScalarResources {
		assigned = min(rQuant, allocatable.ScalarResources[rName]-requested.ScalarResources[rName])
		podRequest.ScalarResources[rName] -= assigned
		res.ScalarResources[rName] = assigned
		if podRequest.ScalarResources[rName] > 0 {
			finished = false
		}
	}

	return res, finished
}

// ResourceListIgnoreZeroResources returns non-zero ResourceList from a framework.Resource.
func ResourceListIgnoreZeroResources(r *framework.Resource) corev1.ResourceList {
	if r == nil {
		return nil
	}
	result := make(corev1.ResourceList)
	if r.MilliCPU > 0 {
		result[corev1.ResourceCPU] = *resource.NewMilliQuantity(r.MilliCPU, resource.DecimalSI)
	}
	if r.Memory > 0 {
		result[corev1.ResourceMemory] = *resource.NewQuantity(r.MilliCPU, resource.BinarySI)
	}
	if r.AllowedPodNumber > 0 {
		result[corev1.ResourcePods] = *resource.NewQuantity(int64(r.AllowedPodNumber), resource.BinarySI)
	}
	if r.EphemeralStorage > 0 {
		result[corev1.ResourceEphemeralStorage] = *resource.NewQuantity(r.EphemeralStorage, resource.BinarySI)
	}
	for rName, rQuant := range r.ScalarResources {
		if rQuant > 0 {
			if v1helper.IsHugePageResourceName(rName) {
				result[rName] = *resource.NewQuantity(rQuant, resource.BinarySI)
			} else {
				result[rName] = *resource.NewQuantity(rQuant, resource.DecimalSI)
			}
		}
	}
	return result
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
