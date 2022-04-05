package dynamic

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gocrane/crane-scheduler/pkg/plugins/apis/policy"
	"k8s.io/klog/v2"

	v1 "k8s.io/api/core/v1"
	framework "k8s.io/kubernetes/pkg/scheduler/framework"

	utils "github.com/gocrane/crane-scheduler/pkg/utils"
)

const (
	// MinTimestampStrLength defines the min length of timestamp string.
	MinTimestampStrLength = 5
	// NodeHotValue is the key of hot value annotation.
	NodeHotValue = "node_hot_value"
	// DefautlHotVauleActivePeriod defines the validity period of nodes' hotvalue.
	DefautlHotVauleActivePeriod = 5 * time.Minute
	// ExtraActivePeriod gives extra active time to the annotation.
	ExtraActivePeriod = 5 * time.Minute
)

// inActivePeriod judges if node annotation with this timestamp is effective.
func inActivePeriod(updatetimeStr string, activeDuration time.Duration) bool {
	if len(updatetimeStr) < MinTimestampStrLength {
		klog.Errorf("[crane] illegel timestamp: %s", updatetimeStr)
		return false
	}

	originUpdateTime, err := time.ParseInLocation(utils.TimeFormat, updatetimeStr, utils.GetLocation())
	if err != nil {
		klog.Errorf("[crane] failed to parse timestamp: %v", err)
		return false
	}

	now, updatetime := time.Now(), originUpdateTime.Add(activeDuration)

	if now.Before(updatetime) {
		return true
	}

	return false
}

func getResourceUsage(anno map[string]string, key string, activeDuration time.Duration) (float64, error) {
	usedstr, ok := anno[key]
	if !ok {
		return 0, fmt.Errorf("key[%s] not found", usedstr)
	}

	usedSlice := strings.Split(usedstr, ",")
	if len(usedSlice) != 2 {
		return 0, fmt.Errorf("illegel value: %s", usedstr)
	}

	if !inActivePeriod(usedSlice[1], activeDuration) {
		return 0, fmt.Errorf("timestamp[%s] is expired", usedstr)
	}

	UsedValue, err := strconv.ParseFloat(usedSlice[0], 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse float[%s]", usedSlice[0])
	}

	if UsedValue < 0 {
		return 0, fmt.Errorf("illegel value: %s", usedstr)
	}

	return UsedValue, nil
}

func getScore(anno map[string]string, priorityPolicy policy.PriorityPolicy, syncPeriod []policy.SyncPolicy) (float64, error) {
	activeDuration, err := getActiveDuration(syncPeriod, priorityPolicy.Name)
	if err != nil || activeDuration == 0 {
		return 0, fmt.Errorf("failed to get the active duration of resource[%s]: %v, while the actual value is %v", priorityPolicy.Name, err, activeDuration)
	}

	usage, err := getResourceUsage(anno, priorityPolicy.Name, activeDuration)
	if err != nil {
		return 0, err
	}

	score := (1. - usage) * priorityPolicy.Weight * float64(framework.MaxNodeScore)

	return score, nil
}

func isOverLoad(name string, anno map[string]string, predicatePolicy policy.PredicatePolicy, activeDuration time.Duration) bool {
	usage, err := getResourceUsage(anno, predicatePolicy.Name, activeDuration)
	if err != nil {
		klog.Errorf("[crane] can not get the usage of resource[%s] from node[%s]'s annotation: %v", predicatePolicy.Name, name, err)
		return false
	}

	// threshold was set as 0 means that the filter according to this metric is useless.
	if predicatePolicy.MaxLimitPecent == 0 {
		klog.V(4).Info("[crane] ignore the filter of resource[%s] for MaxLimitPecent was set as 0")
		return false
	}

	if usage > predicatePolicy.MaxLimitPecent {
		return true
	}

	return false
}

func getNodeScore(name string, anno map[string]string, policySpec policy.PolicySpec) int {

	lenPriorityPolicyList := len(policySpec.Priority)
	if lenPriorityPolicyList == 0 {
		klog.Warningf("[crane] no priority policy exists, all nodes scores 0.")
		return 0
	}

	var score, weight float64

	for _, priorityPolicy := range policySpec.Priority {

		priorityScore, err := getScore(anno, priorityPolicy, policySpec.SyncPeriod)
		if err != nil {
			klog.Errorf("[crane] failed to get node 's score: %v", name, priorityPolicy.Name, score)
		}

		weight += priorityPolicy.Weight
		score += priorityScore
	}

	finnalScore := int(score / weight)

	return finnalScore
}

func getActiveDuration(syncPeriodList []policy.SyncPolicy, name string) (time.Duration, error) {
	for _, period := range syncPeriodList {
		if period.Name == name {
			if period.Period.Duration != 0 {
				return period.Period.Duration + ExtraActivePeriod, nil
			}
		}
	}

	return 0, fmt.Errorf("failed to get the active duration")
}

// isDaemonsetPod judges if this pod belongs to one daemonset workload.
func isDaemonsetPod(pod *v1.Pod) bool {
	for _, ownerRef := range pod.GetOwnerReferences() {
		if ownerRef.Kind == "DaemonSet" {
			return true
		}
	}
	return false
}

func getNodeHotValue(node *v1.Node) float64 {
	anno := node.ObjectMeta.Annotations
	if anno == nil {
		return 0
	}

	hotvalue, err := getResourceUsage(anno, NodeHotValue, DefautlHotVauleActivePeriod)
	if err != nil {
		return 0
	}

	klog.V(4).Infof("[crane] Node[%s]'s hotvalue is %f\n", node.Name, hotvalue)

	return hotvalue
}
