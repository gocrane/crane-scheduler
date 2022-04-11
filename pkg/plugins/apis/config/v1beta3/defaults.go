package v1beta3

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

func addDefaultingFuncs(scheme *runtime.Scheme) error {
	return RegisterDefaults(scheme)
}

func SetDefaults_DynamicArgs(obj *DynamicArgs) {
	if obj.PolicyConfigPath == nil {
		path := "/etc/kubernetes/dynamic-scheduler-policy.yaml"
		obj.PolicyConfigPath = &path
	}
	return
}
