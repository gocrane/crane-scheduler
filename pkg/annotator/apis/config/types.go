package config

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	componentbaseconfig "k8s.io/component-base/config"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AnnotatorConfiguration holds configuration for a node annotator.
type AnnotatorConfiguration struct {
	metav1.TypeMeta
	// BindingHeapSize limits the size of Binding Heap, which stores the lastest
	// pod scheduled imformation.
	BindingHeapSize int32
	// ConcurrentSyncs specified the number of annotator controller workers.
	ConcurrentSyncs int32
	// PolicyConfigPath specified the path of Scheduler Policy File.
	PolicyConfigPath string
	// PrometheusAddr is the address of Prometheus Service.
	PrometheusAddr string
	// clientConnection specifies the kubeconfig file and client connection settings for the proxy
	// server to use when communicating with the apiserver.
	ClientConnection componentbaseconfig.ClientConnectionConfiguration
	// LeaderElection defines the configuration of leader election client.
	LeaderElection componentbaseconfig.LeaderElectionConfiguration
	// DebuggingConfiguration holds configuration for Debugging related features.
	Debugging componentbaseconfig.DebuggingConfiguration
}
