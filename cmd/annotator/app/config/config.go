package config

import (
	prom "github.com/gocrane/crane-scheduler/pkg/annotator/prometheus"
	policy "github.com/gocrane/crane-scheduler/pkg/plugins/apis/policy"
	"k8s.io/client-go/informers"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"

	annotatorconfig "github.com/gocrane/crane-scheduler/pkg/annotator/apis/config"
)

// Config is the main context object for node annotator.
type Config struct {
	// ComponentConfig is the annotator's configuration object.
	ComponentConfig annotatorconfig.AnnotatorConfiguration
	// KubeInformerFactory gives access to kubernetes informers for the controller.
	KubeInformerFactory informers.SharedInformerFactory
	// KubeClient is the general kube client.
	KubeClient clientset.Interface
	// PromClient is used for getting metric data from Prometheus.
	PromClient prom.PromClient
	// Policy is a collection of scheduler policies.
	Policy *policy.DynamicSchedulerPolicy
	// EventRecorder is the event sink
	EventRecorder record.EventRecorder
	// LeaderElectionClient is the client used for leader election
	LeaderElectionClient *clientset.Clientset
}

type completedConfig struct {
	*Config
}

// CompletedConfig same as Config, just to swap private object.
type CompletedConfig struct {
	// Embed a private pointer that cannot be instantiated outside of this package.
	*completedConfig
}

// Complete fills in any fields not set that are required to have valid data. It's mutating the receiver.
func (c *Config) Complete() *CompletedConfig {
	cc := completedConfig{c}

	return &CompletedConfig{&cc}
}
