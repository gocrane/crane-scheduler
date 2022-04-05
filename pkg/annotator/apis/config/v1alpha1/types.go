package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	componentbaseconfigv1alpha1 "k8s.io/component-base/config/v1alpha1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AnnotatorConfiguration holds configuration for a node annotator.
type AnnotatorConfiguration struct {
	metav1.TypeMeta `json:",inline"`
	// BindingHeapSize limits the size of Binding Heap, which stores the lastest
	// pod scheduled imformation.
	BindingHeapSize int32 `json:"bindingHeapSize"`
	// ConcurrentSyncs specified the number of annotator controller workers.
	ConcurrentSyncs int32 `json:"concurrentSyncs"`
	// PolicyConfigPath specified the path of Scheduler Policy File.
	PolicyConfigPath string `json:"policyConfigPath"`
	// PrometheusAddr is the address of Prometheus Service.
	PrometheusAddr string `json:"prometheusAddr"`
	// clientConnection specifies the kubeconfig file and client connection settings for the proxy
	// server to use when communicating with the apiserver.
	ClientConnection componentbaseconfigv1alpha1.ClientConnectionConfiguration `json:"clientConnection"`
	// LeaderElection defines the configuration of leader election client.
	LeaderElection componentbaseconfigv1alpha1.LeaderElectionConfiguration `json:",inline"`
	// DebuggingConfiguration holds configuration for Debugging related features.
	Debugging componentbaseconfigv1alpha1.DebuggingConfiguration `json:",inline"`
}
