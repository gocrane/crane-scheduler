package noderesourcetopology

import (
	"context"
	"encoding/json"
	"math/rand"
	"reflect"
	"strconv"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/kubernetes/pkg/scheduler/framework"

	"github.com/gocrane/api/pkg/generated/clientset/versioned/fake"
	topologyv1alpha1 "github.com/gocrane/api/topology/v1alpha1"
)

var (
	nodeName          = "master"
	hugePageResourceA = corev1.ResourceName(corev1.ResourceHugePagesPrefix + "2Mi")
	nrt               = &topologyv1alpha1.NodeResourceTopology{
		ObjectMeta: metav1.ObjectMeta{Name: nodeName},
		CraneManagerPolicy: topologyv1alpha1.ManagerPolicy{
			CPUManagerPolicy:      topologyv1alpha1.CPUManagerPolicyStatic,
			TopologyManagerPolicy: topologyv1alpha1.TopologyManagerPolicySingleNUMANodePodLevel,
		},
		Reserved: map[corev1.ResourceName]resource.Quantity{
			corev1.ResourceCPU:    resource.MustParse("1"),
			corev1.ResourceMemory: resource.MustParse("1Gi"),
		},
		// node1: 2.5 cpu, 4GiB memory
		// node2: 3.9 cpu, 4GiB memory
		Zones: topologyv1alpha1.ZoneList{
			topologyv1alpha1.Zone{
				Name: "node1",
				Type: topologyv1alpha1.ZoneTypeNode,
				Resources: &topologyv1alpha1.ResourceInfo{
					Allocatable: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceCPU:    resource.MustParse("2.5"),
						corev1.ResourceMemory: resource.MustParse("4Gi"),
					},
				},
			},
			topologyv1alpha1.Zone{
				Name: "node2",
				Type: topologyv1alpha1.ZoneTypeNode,
				Resources: &topologyv1alpha1.ResourceInfo{
					Allocatable: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceCPU:    resource.MustParse("3.9"),
						corev1.ResourceMemory: resource.MustParse("4Gi"),
					},
				},
			},
		},
	}
)

const (
	// CPUTestUnit is 1 CPU
	CPUTestUnit = 1000
	// MemTestUnit is 1 GiB
	MemTestUnit = 1024 * 1024 * 1024
)

type assumedPod struct {
	pod  *corev1.Pod
	zone topologyv1alpha1.ZoneList
}

func newResourcePod(aware bool, result topologyv1alpha1.ZoneList, usage ...framework.Resource) *corev1.Pod {
	pod := newPod(usage...)
	if aware {
		pod.Annotations = map[string]string{
			topologyv1alpha1.AnnotationPodTopologyAwarenessKey: "true",
		}
	}
	if len(result) != 0 {
		podTopologyResultBytes, err := json.Marshal(result)
		if err != nil {
			return pod
		}
		pod.Annotations[topologyv1alpha1.AnnotationPodTopologyResultKey] = string(podTopologyResultBytes)
	}
	return pod
}

func newResourceAssumedPod(result topologyv1alpha1.ZoneList, usage ...framework.Resource) *assumedPod {
	return &assumedPod{
		pod:  newPod(usage...),
		zone: result,
	}
}

type zone struct {
	name   string
	cpu    int64
	memory int64
}

func newZoneList(zones []zone) topologyv1alpha1.ZoneList {
	zoneList := make(topologyv1alpha1.ZoneList, 0, len(zones))
	for _, z := range zones {
		elem := topologyv1alpha1.Zone{
			Name: z.name,
			Type: topologyv1alpha1.ZoneTypeNode,
			Resources: &topologyv1alpha1.ResourceInfo{
				Capacity: make(corev1.ResourceList),
			},
		}
		if z.cpu != 0 {
			elem.Resources.Capacity[corev1.ResourceCPU] = *resource.NewMilliQuantity(z.cpu, resource.DecimalSI)
		}
		if z.memory != 0 {
			elem.Resources.Capacity[corev1.ResourceMemory] = *resource.NewQuantity(z.memory, resource.BinarySI)
		}
		zoneList = append(zoneList, elem)
	}
	return zoneList
}

func newPod(usage ...framework.Resource) *corev1.Pod {
	var containers []corev1.Container
	for _, req := range usage {
		rl := corev1.ResourceList{
			corev1.ResourceCPU:              *resource.NewMilliQuantity(req.MilliCPU, resource.DecimalSI),
			corev1.ResourceMemory:           *resource.NewQuantity(req.Memory, resource.BinarySI),
			corev1.ResourceEphemeralStorage: *resource.NewQuantity(req.EphemeralStorage, resource.BinarySI),
		}
		for rName, rQuant := range req.ScalarResources {
			if rName == hugePageResourceA {
				rl[rName] = *resource.NewQuantity(rQuant, resource.BinarySI)
			} else {
				rl[rName] = *resource.NewQuantity(rQuant, resource.DecimalSI)
			}
		}
		containers = append(containers, corev1.Container{
			Resources: corev1.ResourceRequirements{Requests: rl, Limits: rl},
		})
	}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{UID: types.UID(strconv.Itoa(rand.Int()))},
		Spec: corev1.PodSpec{
			Containers: containers,
		},
	}
	return pod
}

func TestTopologyMatch_Filter(t *testing.T) {
	type args struct {
		pod                    *corev1.Pod
		nodeInfo               *framework.NodeInfo
		nrt                    *topologyv1alpha1.NodeResourceTopology
		assumedPods            []*assumedPod
		topologyAwareResources sets.String
	}
	tests := []struct {
		name string
		args args
		want *framework.Status
	}{
		{
			name: "enough resource of node1 and node2",
			args: args{
				pod: newResourcePod(true, nil, framework.Resource{MilliCPU: CPUTestUnit, Memory: MemTestUnit}),
				nodeInfo: framework.NewNodeInfo(
					newResourcePod(true, newZoneList([]zone{{name: "node1", cpu: 1 * CPUTestUnit}}),
						framework.Resource{MilliCPU: 1 * CPUTestUnit, Memory: 2 * MemTestUnit}),
					newResourcePod(true, newZoneList([]zone{{name: "node2", cpu: 1 * CPUTestUnit}}),
						framework.Resource{MilliCPU: 1 * CPUTestUnit, Memory: 1 * MemTestUnit}),
				),
				nrt:                    nrt,
				topologyAwareResources: sets.NewString(string(corev1.ResourceCPU)),
			},
			want: nil,
		},
		{
			name: "enough resource of node1 and node2 with assumed pods",
			args: args{
				pod:      newResourcePod(true, nil, framework.Resource{MilliCPU: CPUTestUnit, Memory: MemTestUnit}),
				nodeInfo: framework.NewNodeInfo(),
				nrt:      nrt,
				assumedPods: []*assumedPod{
					newResourceAssumedPod(newZoneList([]zone{{name: "node1", cpu: 1 * CPUTestUnit}}),
						framework.Resource{MilliCPU: 1 * CPUTestUnit, Memory: 2 * MemTestUnit}),
					newResourceAssumedPod(newZoneList([]zone{{name: "node2", cpu: 1 * CPUTestUnit}}),
						framework.Resource{MilliCPU: 1 * CPUTestUnit, Memory: 1 * MemTestUnit}),
				},
				topologyAwareResources: sets.NewString(string(corev1.ResourceCPU)),
			},
			want: nil,
		},
		{
			name: "no enough cpu resource",
			args: args{
				pod: newResourcePod(true, nil, framework.Resource{MilliCPU: CPUTestUnit, Memory: MemTestUnit}),
				nodeInfo: framework.NewNodeInfo(
					newResourcePod(true, newZoneList([]zone{{name: "node1", cpu: 2 * CPUTestUnit}}),
						framework.Resource{MilliCPU: 2 * CPUTestUnit, Memory: 2 * MemTestUnit}),
					newResourcePod(true, newZoneList([]zone{{name: "node2", cpu: 4 * CPUTestUnit}}),
						framework.Resource{MilliCPU: 4 * CPUTestUnit, Memory: 1 * MemTestUnit}),
				),
				nrt:                    nrt,
				topologyAwareResources: sets.NewString(string(corev1.ResourceCPU)),
			},
			want: framework.NewStatus(framework.Unschedulable, ErrReasonNUMAResourceNotEnough),
		},
		{
			name: "no enough cpu resource in one NUMA node",
			args: args{
				pod: newResourcePod(true, nil, framework.Resource{MilliCPU: 2 * CPUTestUnit, Memory: MemTestUnit}),
				nodeInfo: framework.NewNodeInfo(
					newResourcePod(true, newZoneList([]zone{{name: "node1", cpu: 1 * CPUTestUnit}}),
						framework.Resource{MilliCPU: 1 * CPUTestUnit, Memory: 2 * MemTestUnit}),
					newResourcePod(true, newZoneList([]zone{{name: "node2", cpu: 3 * CPUTestUnit}}),
						framework.Resource{MilliCPU: 3 * CPUTestUnit, Memory: 1 * MemTestUnit}),
				),
				nrt:                    nrt,
				topologyAwareResources: sets.NewString(string(corev1.ResourceCPU)),
			},
			want: framework.NewStatus(framework.Unschedulable, ErrReasonNUMAResourceNotEnough),
		},
		{
			name: "no enough cpu resource in one NUMA node consider assumed pods",
			args: args{
				pod: newResourcePod(true, nil, framework.Resource{MilliCPU: 2 * CPUTestUnit, Memory: MemTestUnit}),
				nodeInfo: framework.NewNodeInfo(
					newResourcePod(true, newZoneList([]zone{{name: "node1", cpu: 1 * CPUTestUnit}}),
						framework.Resource{MilliCPU: 1 * CPUTestUnit, Memory: 2 * MemTestUnit}),
				),
				assumedPods: []*assumedPod{
					newResourceAssumedPod(newZoneList([]zone{{name: "node2", cpu: 3 * CPUTestUnit}}),
						framework.Resource{MilliCPU: 3 * CPUTestUnit, Memory: 1 * MemTestUnit}),
				},
				nrt:                    nrt,
				topologyAwareResources: sets.NewString(string(corev1.ResourceCPU)),
			},
			want: framework.NewStatus(framework.Unschedulable, ErrReasonNUMAResourceNotEnough),
		},
		{
			name: "no enough memory resource in one NUMA node",
			args: args{
				pod: newResourcePod(true, nil, framework.Resource{MilliCPU: 2 * CPUTestUnit, Memory: 2 * MemTestUnit}),
				nodeInfo: framework.NewNodeInfo(
					newResourcePod(true, newZoneList([]zone{{name: "node1", cpu: 1 * CPUTestUnit, memory: 3 * MemTestUnit}}),
						framework.Resource{MilliCPU: 1 * CPUTestUnit, Memory: 3 * MemTestUnit}),
				),
				assumedPods: []*assumedPod{
					newResourceAssumedPod(newZoneList([]zone{{name: "node2", cpu: 1 * CPUTestUnit, memory: 3 * MemTestUnit}}),
						framework.Resource{MilliCPU: 1 * CPUTestUnit, Memory: 3 * MemTestUnit}),
				},
				nrt:                    nrt,
				topologyAwareResources: sets.NewString(string(corev1.ResourceCPU), string(corev1.ResourceMemory)),
			},
			want: framework.NewStatus(framework.Unschedulable, ErrReasonNUMAResourceNotEnough),
		},
		{
			name: "crane agent policy is not static",
			args: args{
				pod: newResourcePod(true, nil, framework.Resource{MilliCPU: CPUTestUnit, Memory: MemTestUnit}),
				nodeInfo: framework.NewNodeInfo(
					newResourcePod(true, newZoneList([]zone{{name: "node1", cpu: 1 * CPUTestUnit}}),
						framework.Resource{MilliCPU: 1 * CPUTestUnit, Memory: 2 * MemTestUnit}),
					newResourcePod(true, newZoneList([]zone{{name: "node2", cpu: 1 * CPUTestUnit}}),
						framework.Resource{MilliCPU: 1 * CPUTestUnit, Memory: 1 * MemTestUnit}),
				),
				nrt: func() *topologyv1alpha1.NodeResourceTopology {
					nrtCopy := nrt.DeepCopy()
					nrtCopy.CraneManagerPolicy.CPUManagerPolicy = topologyv1alpha1.CPUManagerPolicyNone
					return nrtCopy
				}(),
				topologyAwareResources: sets.NewString(string(corev1.ResourceCPU)),
			},
			want: nil,
		},
		{
			name: "no enough cpu resource in one NUMA node with default single numa topology manager policy",
			args: args{
				pod: newResourcePod(false, nil, framework.Resource{MilliCPU: 2 * CPUTestUnit, Memory: MemTestUnit}),
				nodeInfo: framework.NewNodeInfo(
					newResourcePod(true, newZoneList([]zone{{name: "node1", cpu: 1 * CPUTestUnit}}),
						framework.Resource{MilliCPU: 1 * CPUTestUnit, Memory: 2 * MemTestUnit}),
					newResourcePod(true, newZoneList([]zone{{name: "node2", cpu: 3 * CPUTestUnit}}),
						framework.Resource{MilliCPU: 3 * CPUTestUnit, Memory: 1 * MemTestUnit}),
				),
				nrt:                    nrt,
				topologyAwareResources: sets.NewString(string(corev1.ResourceCPU)),
			},
			want: framework.NewStatus(framework.Unschedulable, ErrReasonNUMAResourceNotEnough),
		},
		{
			name: "enough cpu resource in node with default none topology manager policy",
			args: args{
				pod: newResourcePod(false, nil, framework.Resource{MilliCPU: 2 * CPUTestUnit, Memory: MemTestUnit}),
				nodeInfo: framework.NewNodeInfo(
					newResourcePod(true, newZoneList([]zone{{name: "node1", cpu: 1 * CPUTestUnit}}),
						framework.Resource{MilliCPU: 1 * CPUTestUnit, Memory: 2 * MemTestUnit}),
					newResourcePod(true, newZoneList([]zone{{name: "node2", cpu: 3 * CPUTestUnit}}),
						framework.Resource{MilliCPU: 3 * CPUTestUnit, Memory: 1 * MemTestUnit}),
				),
				nrt: func() *topologyv1alpha1.NodeResourceTopology {
					nrtCopy := nrt.DeepCopy()
					nrtCopy.CraneManagerPolicy.TopologyManagerPolicy = topologyv1alpha1.TopologyManagerPolicyNone
					return nrtCopy
				}(),
				topologyAwareResources: sets.NewString(string(corev1.ResourceCPU)),
			},
			want: nil,
		},
		{
			name: "no enough cpu resource in one NUMA node with default single numa topology manager policy",
			args: args{
				pod: newResourcePod(false, nil, framework.Resource{MilliCPU: 2 * CPUTestUnit, Memory: MemTestUnit}),
				nodeInfo: framework.NewNodeInfo(
					newResourcePod(true, newZoneList([]zone{{name: "node1", cpu: 1 * CPUTestUnit}}),
						framework.Resource{MilliCPU: 1 * CPUTestUnit, Memory: 2 * MemTestUnit}),
					newResourcePod(true, newZoneList([]zone{{name: "node2", cpu: 3 * CPUTestUnit}}),
						framework.Resource{MilliCPU: 3 * CPUTestUnit, Memory: 1 * MemTestUnit}),
				),
				nrt:                    nrt,
				topologyAwareResources: sets.NewString(string(corev1.ResourceCPU)),
			},
			want: framework.NewStatus(framework.Unschedulable, ErrReasonNUMAResourceNotEnough),
		},
		{
			name: "enough cpu resource in one NUMA node with cross numa pods",
			args: args{
				pod: newResourcePod(false, nil, framework.Resource{MilliCPU: 2 * CPUTestUnit, Memory: MemTestUnit}),
				nodeInfo: framework.NewNodeInfo(
					newResourcePod(true, newZoneList([]zone{{name: "node1", cpu: 1 * CPUTestUnit}}),
						framework.Resource{MilliCPU: 1 * CPUTestUnit, Memory: 2 * MemTestUnit}),
					newResourcePod(true, newZoneList([]zone{{name: "node1", cpu: 1 * CPUTestUnit}, {name: "node2", cpu: 1 * CPUTestUnit}}),
						framework.Resource{MilliCPU: 2 * CPUTestUnit, Memory: 1 * MemTestUnit}),
				),
				nrt:                    nrt,
				topologyAwareResources: sets.NewString(string(corev1.ResourceCPU)),
			},
			want: nil,
		},
		{
			name: "no enough cpu resource in one NUMA node with cross numa pods",
			args: args{
				pod: newResourcePod(false, nil, framework.Resource{MilliCPU: 2 * CPUTestUnit, Memory: MemTestUnit}),
				nodeInfo: framework.NewNodeInfo(
					newResourcePod(true, newZoneList([]zone{{name: "node1", cpu: 1 * CPUTestUnit}}),
						framework.Resource{MilliCPU: 1 * CPUTestUnit, Memory: 2 * MemTestUnit}),
					newResourcePod(true, newZoneList([]zone{{name: "node1", cpu: 1 * CPUTestUnit}, {name: "node2", cpu: 2 * CPUTestUnit}}),
						framework.Resource{MilliCPU: 3 * CPUTestUnit, Memory: 1 * MemTestUnit}),
				),
				nrt:                    nrt,
				topologyAwareResources: sets.NewString(string(corev1.ResourceCPU)),
			},
			want: framework.NewStatus(framework.Unschedulable, ErrReasonNUMAResourceNotEnough),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			fakeClient := fake.NewSimpleClientset(tt.args.nrt)
			lister, err := initTopologyInformer(ctx, fakeClient)
			if err != nil {
				t.Fatalf("initTopologyInformer function error: %v", err)
			}
			node := corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: nodeName},
			}
			tt.args.nodeInfo.SetNode(&node)

			cache := NewPodTopologyCache(ctx, 30*time.Second)
			for _, aps := range tt.args.assumedPods {
				tt.args.nodeInfo.AddPod(aps.pod)
				if err := cache.AssumePod(aps.pod, aps.zone); err != nil {
					t.Errorf("assume pod error: %v", err)
				}
			}

			var p framework.Plugin = &TopologyMatch{
				lister:                 lister,
				PodTopologyCache:       cache,
				topologyAwareResources: tt.args.topologyAwareResources,
			}
			cycleState := framework.NewCycleState()
			_, preFilterStatus := p.(framework.PreFilterPlugin).PreFilter(ctx, cycleState, tt.args.pod)
			if !preFilterStatus.IsSuccess() {
				t.Errorf("prefilter failed with status: %v", preFilterStatus)
			}
			gotStatus := p.(framework.FilterPlugin).Filter(ctx, cycleState, tt.args.pod, tt.args.nodeInfo)
			if !reflect.DeepEqual(gotStatus, tt.want) {
				t.Errorf("status does not match: %v, want: %v", gotStatus, tt.want)
			}

		})
	}
}
