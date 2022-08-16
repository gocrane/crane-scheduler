package config

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DynamicArgs is the args struction of Dynamic scheduler plugin.
type DynamicArgs struct {
	metav1.TypeMeta
	// PolicyConfigPath specified the path of policy config.
	PolicyConfigPath string
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeResourceTopologyMatchArgs holds arguments used to configure the NodeResourceTopologyMatch plugin.
type NodeResourceTopologyMatchArgs struct {
	metav1.TypeMeta
	// TopologyAwareResources represents the resource names of topology.
	TopologyAwareResources []string
}
