package policy

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type DynamicSchedulerPolicy struct {
	metav1.TypeMeta
	Spec PolicySpec
}

type PolicySpec struct {
	SyncPeriod []SyncPolicy
	Predicate  []PredicatePolicy
	Priority   []PriorityPolicy
	HotValue   []HotValuePolicy
}

type SyncPolicy struct {
	Name   string
	Period metav1.Duration
}

type PredicatePolicy struct {
	Name           string
	MaxLimitPecent float64
}

type PriorityPolicy struct {
	Name   string
	Weight float64
}

type HotValuePolicy struct {
	TimeRange metav1.Duration
	Count     int
}
