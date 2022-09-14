package noderesourcetopology

import (
	"context"
	"encoding/json"
	"fmt"

	jsonpatch "github.com/evanphx/json-patch"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/scheduler/framework"

	topologyv1alpha1 "github.com/gocrane/api/topology/v1alpha1"
)

// PreBind writes pod topology result annotations using the k8s client.
func (tm *TopologyMatch) PreBind(
	ctx context.Context,
	state *framework.CycleState,
	pod *corev1.Pod,
	nodeName string,
) *framework.Status {
	klog.V(4).InfoS("Attempting to prebind pod to node", "pod", klog.KObj(pod), "node", nodeName)
	s, err := getStateData(state)
	if err != nil {
		return framework.AsStatus(err)
	}

	if len(s.topologyResult) == 0 {
		return nil
	}

	result, err := json.Marshal(s.topologyResult)
	if err != nil {
		return framework.AsStatus(err)
	}
	newObj := pod.DeepCopy()
	if newObj.Annotations == nil {
		newObj.Annotations = make(map[string]string)
	}
	newObj.Annotations[topologyv1alpha1.AnnotationPodTopologyResultKey] = string(result)

	oldData, err := json.Marshal(pod)
	if err != nil {
		return framework.AsStatus(err)
	}
	newData, err := json.Marshal(newObj)
	if err != nil {
		return framework.AsStatus(err)
	}

	patchBytes, err := jsonpatch.CreateMergePatch(oldData, newData)
	if err != nil {
		return framework.AsStatus(fmt.Errorf("failed to create merge patch: %v", err))
	}

	_, err = tm.handle.ClientSet().CoreV1().Pods(pod.Namespace).Patch(ctx, pod.Name,
		types.MergePatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		return framework.AsStatus(err)
	}
	return nil
}
