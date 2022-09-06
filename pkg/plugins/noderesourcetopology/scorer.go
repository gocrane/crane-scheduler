package noderesourcetopology

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

// Score invoked at the Score extension point.
func (tm *TopologyMatch) Score(
	ctx context.Context,
	state *framework.CycleState,
	pod *corev1.Pod,
	nodeName string,
) (int64, *framework.Status) {
	s, err := getStateData(state)
	if err != nil {
		return 0, framework.AsStatus(err)
	}

	nw, exist := s.podTopologyByNode[nodeName]
	if !exist {
		return 0, nil
	}

	// TODO(Garrybest): make this plugin as configurable.
	return framework.MaxNodeScore / int64(len(nw.result)), nil
}

// ScoreExtensions of the Score plugin.
func (tm *TopologyMatch) ScoreExtensions() framework.ScoreExtensions {
	return nil
}
