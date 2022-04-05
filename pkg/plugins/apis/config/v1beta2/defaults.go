package v1beta2

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

func addDefaultingFuncs(scheme *runtime.Scheme) error {
	return RegisterDefaults(scheme)
}

func SetDefaults_DynamicArgs(obj *DynamicArgs) {
	if obj.PolicyConfigPath == "" {
		obj.PolicyConfigPath = "/etc/kubernetes/dynamic-scheduler-policy.yaml"
	}
	return
}
