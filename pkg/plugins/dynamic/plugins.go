package dynamic

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/scheduler/framework"

	"github.com/gocrane/crane-scheduler/pkg/plugins/apis/config"
	"github.com/gocrane/crane-scheduler/pkg/plugins/apis/policy"
	"github.com/gocrane/crane-scheduler/pkg/utils"
)

var _ framework.FilterPlugin = &DynamicScheduler{}
var _ framework.ScorePlugin = &DynamicScheduler{}

const (
	// Name is the name of the plugin used in the plugin registry and configurations.
	Name = "Dynamic"
)

// Dynamic-scheduler is a real load-aware scheduler plugin.
type DynamicScheduler struct {
	handle          framework.Handle
	schedulerPolicy *policy.DynamicSchedulerPolicy
}

// Name returns name of the plugin.
func (ds *DynamicScheduler) Name() string {
	return Name
}

// Filter invoked at the filter extension point.
// checkes if the real load of one node is too high.
// It returns a list of failure reasons if the node is overload.
func (ds *DynamicScheduler) Filter(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeInfo *framework.NodeInfo) *framework.Status {
	// ignore daemonset pod
	if utils.IsDaemonsetPod(pod) {
		return framework.NewStatus(framework.Success, "")
	}

	node := nodeInfo.Node()
	if node == nil {
		return framework.NewStatus(framework.Error, "node not found")
	}

	nodeAnnotations, nodeName := nodeInfo.Node().Annotations, nodeInfo.Node().Name
	if nodeAnnotations == nil {
		nodeAnnotations = map[string]string{}
	}

	for _, policy := range ds.schedulerPolicy.Spec.Predicate {
		activeDuration, err := getActiveDuration(ds.schedulerPolicy.Spec.SyncPeriod, policy.Name)

		if err != nil || activeDuration == 0 {
			klog.Warningf("[crane] failed to get active duration: %v", err)
			continue
		}

		if isOverLoad(nodeName, nodeAnnotations, policy, activeDuration) {
			return framework.NewStatus(framework.Unschedulable, fmt.Sprintf("Load[%s] of node[%s] is too high", policy.Name, nodeName))
		}

	}
	return framework.NewStatus(framework.Success, "")
}

// Score invoked at the Score extension point.
// It gets metric data from node annotation, and favors nodes with the least real resource usage.
func (ds *DynamicScheduler) Score(ctx context.Context, state *framework.CycleState, p *v1.Pod, nodeName string) (int64, *framework.Status) {
	nodeInfo, err := ds.handle.SnapshotSharedLister().NodeInfos().Get(nodeName)
	if err != nil {
		return 0, framework.NewStatus(framework.Error, fmt.Sprintf("getting node %q from Snapshot: %v", nodeName, err))
	}

	node := nodeInfo.Node()
	if node == nil {
		return 0, framework.NewStatus(framework.Error, "node not found")
	}

	nodeAnnotations := node.Annotations
	if nodeAnnotations == nil {
		nodeAnnotations = map[string]string{}
	}

	score, hotValue := getNodeScore(node.Name, nodeAnnotations, ds.schedulerPolicy.Spec), getNodeHotValue(node)

	score = score - int(hotValue*10)

	finalScore := utils.NormalizeScore(int64(score),framework.MaxNodeScore,framework.MinNodeScore)

	klog.V(4).Infof("[crane] Node[%s]'s final score is %d, while score is %d and hot value is %f", node.Name, finalScore, score, hotValue)

	return finalScore, nil
}

func (ds *DynamicScheduler) ScoreExtensions() framework.ScoreExtensions {
	return nil
}

// NewDynamicScheduler returns a Crane Scheduler object.
func NewDynamicScheduler(plArgs runtime.Object, h framework.Handle) (framework.Plugin, error) {
	args, ok := plArgs.(*config.DynamicArgs)
	if !ok {
		return nil, fmt.Errorf("want args to be of type DynamicArgs, got %T.", plArgs)
	}

	schedulerPolicy, err := LoadPolicyFromFile(args.PolicyConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get scheduler policy from config file: %v", err)
	}

	return &DynamicScheduler{
		schedulerPolicy: schedulerPolicy,
		handle:          h,
	}, nil
}
