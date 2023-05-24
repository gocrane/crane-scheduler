package noderesourcetopology

import (
	"context"
	"reflect"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/kubernetes/pkg/scheduler/framework"

	"github.com/gocrane/api/pkg/generated/clientset/versioned/fake"
	topologyv1alpha1 "github.com/gocrane/api/topology/v1alpha1"
)

func TestTopologyMatch_Score(t *testing.T) {
	type args struct {
		pod                    *corev1.Pod
		nodeInfo               *framework.NodeInfo
		nrt                    *topologyv1alpha1.NodeResourceTopology
		assumedPods            []*assumedPod
		topologyAwareResources sets.String
	}
	type res struct {
		score  int64
		status *framework.Status
	}
	tests := []struct {
		name string
		args args
		want res
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
			want: res{
				score:  100,
				status: nil,
			},
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
			want: res{
				score:  100,
				status: nil,
			},
		},
		{
			name: "enough cpu resource in one NUMA node with cross numa pods",
			args: args{
				pod: newResourcePod(false, nil, framework.Resource{MilliCPU: 2 * CPUTestUnit, Memory: MemTestUnit}),
				nodeInfo: framework.NewNodeInfo(
					newResourcePod(true, newZoneList([]zone{{name: "node1", cpu: 1 * CPUTestUnit}, {name: "node2", cpu: 1 * CPUTestUnit}}),
						framework.Resource{MilliCPU: 2 * CPUTestUnit, Memory: 2 * MemTestUnit}),
					newResourcePod(true, newZoneList([]zone{{name: "node2", cpu: 1 * CPUTestUnit}}),
						framework.Resource{MilliCPU: 1 * CPUTestUnit, Memory: 1 * MemTestUnit}),
				),
				nrt: func() *topologyv1alpha1.NodeResourceTopology {
					nrtCopy := nrt.DeepCopy()
					nrtCopy.CraneManagerPolicy.TopologyManagerPolicy = topologyv1alpha1.TopologyManagerPolicyNone
					return nrtCopy
				}(),
				topologyAwareResources: sets.NewString(string(corev1.ResourceCPU)),
			},
			want: res{
				score:  50,
				status: nil,
			},
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
			filterStatus := p.(framework.FilterPlugin).Filter(ctx, cycleState, tt.args.pod, tt.args.nodeInfo)
			if !filterStatus.IsSuccess() {
				t.Errorf("filter failed with status: %v", preFilterStatus)
			}
			score, gotStatus := p.(framework.ScorePlugin).Score(ctx, cycleState, tt.args.pod, tt.args.nodeInfo.Node().Name)
			if !reflect.DeepEqual(res{score, gotStatus}, tt.want) {
				t.Errorf("status does not match: %v, want: %v", gotStatus, tt.want)
			}
		})
	}
}
