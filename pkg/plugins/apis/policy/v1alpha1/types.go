package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type DynamicSchedulerPolicy struct {
	metav1.TypeMeta `json:",inline"`
	Spec            PolicySpec `json:"spec"`
}

type PolicySpec struct {
	SyncPeriod []SyncPolicy      `json:"syncPolicy"`
	Predicate  []PredicatePolicy `json:"predicate"`
	Priority   []PriorityPolicy  `json:"priority"`
	HotValue   []HotValuePolicy  `json:"hotValue"`
}

type SyncPolicy struct {
	Name   string          `json:"name"`
	Period metav1.Duration `json:"period"`
}

type PredicatePolicy struct {
	Name           string  `json:"name"`
	MaxLimitPecent float64 `json:"maxLimitPecent"`
}

type PriorityPolicy struct {
	Name   string  `json:"name"`
	Weight float64 `json:"weight"`
}

type HotValuePolicy struct {
	TimeRange metav1.Duration `json:"timeRange"`
	Count     int             `json:"count"`
}
