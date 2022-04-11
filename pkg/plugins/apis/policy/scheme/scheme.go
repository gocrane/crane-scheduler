package scheme

import (
	dynamicpolicy "github.com/gocrane/crane-scheduler/pkg/plugins/apis/policy"
	dynamicpolicyv1alpha1 "github.com/gocrane/crane-scheduler/pkg/plugins/apis/policy/v1alpha1"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

var (
	// Scheme is the runtime.Scheme to which all kubescheduler api types are registered.
	Scheme = runtime.NewScheme()

	// Codecs provides access to encoding and decoding for the scheme.
	Codecs = serializer.NewCodecFactory(Scheme, serializer.EnableStrict)
)

func init() {
	AddToScheme(Scheme)
}

// AddToScheme builds the kubescheduler scheme using all known versions of the kubescheduler api.
func AddToScheme(scheme *runtime.Scheme) {
	utilruntime.Must(dynamicpolicy.AddToScheme(scheme))
	utilruntime.Must(dynamicpolicyv1alpha1.AddToScheme(scheme))
}
