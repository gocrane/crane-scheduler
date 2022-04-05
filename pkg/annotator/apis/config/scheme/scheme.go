package scheme

import (
	annotatorconfig "github.com/gocrane/crane-scheduler/pkg/annotator/apis/config"
	annotatorconfigv1alpha1 "github.com/gocrane/crane-scheduler/pkg/annotator/apis/config/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

var (
	// Scheme is the runtime.Scheme to which all annotator api types are registered.
	Scheme = runtime.NewScheme()

	// Codecs provides access to encoding and decoding for the scheme.
	// Codecs = serializer.NewCodecFactory(Scheme, serializer.EnableStrict)
	Codecs = serializer.NewCodecFactory(Scheme)
)

func init() {
	AddToScheme(Scheme)
}

// AddToScheme builds the annotator scheme using all known versions of the kubescheduler api.
func AddToScheme(scheme *runtime.Scheme) {
	utilruntime.Must(annotatorconfig.AddToScheme(scheme))
	utilruntime.Must(annotatorconfigv1alpha1.AddToScheme(scheme))
}
