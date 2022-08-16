package v1beta2

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kubeschedulerschemev1beta2 "k8s.io/kube-scheduler/config/v1beta2"
	kubeschedulerconfig "k8s.io/kubernetes/pkg/scheduler/apis/config"
)

// SchemeGroupVersion is group version used to register these objects
var SchemeGroupVersion = schema.GroupVersion{Group: kubeschedulerconfig.GroupName, Version: "v1beta2"}

var (
	// localSchemeBuilder and AddToScheme will stay in k8s.io/kubernetes.
	localSchemeBuilder = &kubeschedulerschemev1beta2.SchemeBuilder
	// AddToScheme is a global function that registers this API group & version to a scheme
	AddToScheme = localSchemeBuilder.AddToScheme
)

// addKnownTypes registers known types to the given scheme
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&DynamicArgs{},
		&NodeResourceTopologyMatchArgs{},
	)
	return nil
}

func init() {
	// We only register manually written functions here. The registration of the
	// generated functions takes place in the generated files. The separation
	// makes the code compile even when the generated files are missing.
	localSchemeBuilder.Register(addKnownTypes)
	localSchemeBuilder.Register(RegisterDefaults)
	localSchemeBuilder.Register(RegisterConversions)
}
