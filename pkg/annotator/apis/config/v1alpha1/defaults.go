package v1alpha1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
	componentbaseconfigv1alpha1 "k8s.io/component-base/config/v1alpha1"
)

func addDefaultingFuncs(scheme *runtime.Scheme) error {
	return RegisterDefaults(scheme)
}

func SetDefaults_AnnotatorConfiguration(obj *AnnotatorConfiguration) {
	if obj.BindingHeapSize == 0 {
		obj.BindingHeapSize = 1024
	}

	if obj.ConcurrentSyncs == 0 {
		obj.ConcurrentSyncs = 1
	}

	if obj.PolicyConfigPath == "" {
		obj.PolicyConfigPath = "/data/policy.yaml"
	}

	componentbaseconfigv1alpha1.RecommendedDefaultLeaderElectionConfiguration(&obj.LeaderElection)
	componentbaseconfigv1alpha1.RecommendedDebuggingConfiguration(&obj.Debugging)
}
