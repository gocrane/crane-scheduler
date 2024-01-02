package v1beta2

import (
	"k8s.io/apimachinery/pkg/runtime"
	kubeschedulerscheme "k8s.io/kube-scheduler/config/v1"
)

// SchemeGroupVersion is group version used to register these objects
var SchemeGroupVersion = kubeschedulerscheme.SchemeGroupVersion

var (
	// localSchemeBuilder and AddToScheme will stay in k8s.io/kubernetes.
	localSchemeBuilder = &kubeschedulerscheme.SchemeBuilder
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
