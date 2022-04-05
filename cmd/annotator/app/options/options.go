package options

import (
	"fmt"

	"github.com/gocrane/crane-scheduler/pkg/annotator/prometheus"
	"github.com/spf13/pflag"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	options "k8s.io/component-base/config/options"

	annotatorappconfig "github.com/gocrane/crane-scheduler/cmd/annotator/app/config"
	annotatorconfig "github.com/gocrane/crane-scheduler/pkg/annotator/apis/config"
	annotatorconfigscheme "github.com/gocrane/crane-scheduler/pkg/annotator/apis/config/scheme"
	annotatorconfigv1alpha1 "github.com/gocrane/crane-scheduler/pkg/annotator/apis/config/v1alpha1"

	dynamicscheduler "github.com/gocrane/crane-scheduler/pkg/plugins/dynamic"
)

const (
	AnnotatorControllerUserAgent = "annotator"
)

// Options has all the params needed to run a Annotator.
type Options struct {
	*annotatorconfig.AnnotatorConfiguration

	master string
}

// NewOptions returns default annotator app options.
func NewOptions() (*Options, error) {
	componentConfig, err := newDefaultComponentConfig()
	if err != nil {
		return nil, err
	}

	o := &Options{
		AnnotatorConfiguration: componentConfig,
	}

	o.LeaderElection.ResourceName = "annotator"
	o.LeaderElection.ResourceNamespace = "kube-system"

	return o, nil
}

func newDefaultComponentConfig() (*annotatorconfig.AnnotatorConfiguration, error) {
	versioned := &annotatorconfigv1alpha1.AnnotatorConfiguration{}
	annotatorconfigscheme.Scheme.Default(versioned)

	internal := &annotatorconfig.AnnotatorConfiguration{}
	if err := annotatorconfigscheme.Scheme.Convert(versioned, internal, nil); err != nil {
		return nil, err
	}

	return internal, nil
}

// Flags returns flags for a specific Annotator by section name.
func (o *Options) Flags(flag *pflag.FlagSet) error {
	if flag == nil {
		return fmt.Errorf("nil pointer")
	}

	flag.StringVar(&o.PolicyConfigPath, "policy-config-path", o.PolicyConfigPath, "Path to annotator policy cofig")
	flag.StringVar(&o.PrometheusAddr, "prometheus-addr", o.PrometheusAddr, "The address of prometheus, from which we can pull metrics data.")
	flag.Int32Var(&o.BindingHeapSize, "binding-heap-size", o.BindingHeapSize, "Max size of binding heap size, used to store hot value data.")
	flag.Int32Var(&o.ConcurrentSyncs, "concurrent-syncs", o.ConcurrentSyncs, "The number of annotator controller workers that are allowed to sync concurrently.")
	flag.StringVar(&o.ClientConnection.Kubeconfig, "kubeconfig", o.ClientConnection.Kubeconfig, "Path to kubeconfig file with authorization information")
	flag.StringVar(&o.master, "master", o.master, "The address of the Kubernetes API server (overrides any value in kubeconfig)")

	options.BindLeaderElectionFlags(&o.LeaderElection, flag)
	return nil
}

// ApplyTo fills up Annotator config with options.
func (o *Options) ApplyTo(c *annotatorappconfig.Config) error {
	c.ComponentConfig = *o.AnnotatorConfiguration
	return nil
}

// Validate validates the options and config before launching Annotator.
func (o *Options) Validate() error {
	return nil
}

// Config returns an Annotator config object.
func (o *Options) Config() (*annotatorappconfig.Config, error) {
	var kubeconfig *rest.Config
	var err error

	if err := o.Validate(); err != nil {
		return nil, err
	}

	c := &annotatorappconfig.Config{}
	if err := o.ApplyTo(c); err != nil {
		return nil, err
	}

	c.Policy, err = dynamicscheduler.LoadPolicyFromFile(o.PolicyConfigPath)
	if err != nil {
		return nil, err
	}

	if o.ClientConnection.Kubeconfig == "" {
		kubeconfig, err = rest.InClusterConfig()
	} else {
		// Build config from configfile
		kubeconfig, err = clientcmd.BuildConfigFromFlags(o.master, o.ClientConnection.Kubeconfig)
	}
	if err != nil {
		return nil, err
	}

	c.KubeClient, err = clientset.NewForConfig(rest.AddUserAgent(kubeconfig, AnnotatorControllerUserAgent))
	if err != nil {
		return nil, err
	}

	c.LeaderElectionClient = clientset.NewForConfigOrDie(rest.AddUserAgent(kubeconfig, "leader-election"))

	c.PromClient, err = prometheus.NewPromClient(o.PrometheusAddr)
	if err != nil {
		return nil, err
	}

	c.KubeInformerFactory = NewInformerFactory(c.KubeClient, 0)

	return c, nil
}
