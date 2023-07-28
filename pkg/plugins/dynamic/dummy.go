package dynamic

import (
	"context"
	"fmt"
	"github.com/gocrane/crane-scheduler/pkg/plugins/apis/policy"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"k8s.io/kubernetes/pkg/scheduler/util"
)

var _ framework.FilterPlugin = &DummyScheduler{}
var _ framework.PostFilterPlugin = &DummyScheduler{}

const (
	// SchedulerName is the name of the plugin used in the plugin registry and configurations.
	SchedulerName = "Dummy"
	stateKey      = "DUMMY_DATA"
)

var _ framework.StateData = &stateData{}

type stateData struct {
	podName  string
	nodeName string
}

func (s *stateData) Clone() framework.StateData {
	return s
}

type DummyScheduler struct {
	handle          framework.Handle
	schedulerPolicy *policy.DynamicSchedulerPolicy
}

// Name returns name of the plugin.
func (ds *DummyScheduler) Name() string {
	return Name
}

func (ds *DummyScheduler) Filter(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeInfo *framework.NodeInfo) *framework.Status {
	if pod.Name == "nginx" && pod.Status.NominatedNodeName == "" {
		return framework.NewStatus(framework.Unschedulable, fmt.Sprintf("reject pod to trigger PostFilter %v", pod.Name))
	}

	return nil
}

func (ds *DummyScheduler) PostFilter(ctx context.Context, state *framework.CycleState, pod *v1.Pod, filteredNodeStatusMap framework.NodeToStatusMap) (*framework.PostFilterResult, *framework.Status) {
	cs := ds.handle.ClientSet()
	victim, err := cs.CoreV1().Pods("default").Get(context.TODO(), "dummy", metav1.GetOptions{})
	if err != nil {
		klog.ErrorS(err, "Get pod", "pod", "dummy", "preemptor", klog.KObj(pod))
		return nil, framework.AsStatus(err)
	}

	if err := util.DeletePod(cs, victim); err != nil {
		klog.ErrorS(err, "Preempting pod", "pod", klog.KObj(victim), "preemptor", klog.KObj(pod))
		return nil, framework.AsStatus(err)
	}
	return framework.NewPostFilterResultWithNominatedNode("10.0.0.36"), framework.NewStatus(framework.Success,
		fmt.Sprintf("PostFilter use node %s for pod %s.", "10.0.0.36", pod.Name))
}

func (ds *DynamicScheduler) DummyScheduler() framework.ScoreExtensions {
	return nil
}

// NewDummyScheduler returns a Crane Scheduler object.
func NewDummyScheduler(plArgs runtime.Object, h framework.Handle) (framework.Plugin, error) {
	return &DummyScheduler{
		handle: h,
	}, nil
}
