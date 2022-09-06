package noderesourcetopology

import (
	"context"
	"fmt"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/scheduler/framework"

	topologyclientset "github.com/gocrane/api/pkg/generated/clientset/versioned"
	informers "github.com/gocrane/api/pkg/generated/informers/externalversions"
	listerv1alpha1 "github.com/gocrane/api/pkg/generated/listers/topology/v1alpha1"
	topologyv1alpha1 "github.com/gocrane/api/topology/v1alpha1"

	"github.com/gocrane/crane-scheduler/pkg/plugins/apis/config"
)

const (
	// Name is the name of the plugin used in the plugin registry and configurations.
	Name = "NodeResourceTopologyMatch"

	// stateKey is the key in CycleState to NodeResourcesTopology.
	stateKey framework.StateKey = Name
)

// New initializes a new plugin and returns it.
func New(args runtime.Object, handle framework.Handle) (framework.Plugin, error) {
	klog.V(2).InfoS("Creating new TopologyMatch plugin")
	cfg, ok := args.(*config.NodeResourceTopologyMatchArgs)
	if !ok {
		return nil, fmt.Errorf("want args to be of type NodeResourceTopologyMatchArgs, got %T", args)
	}

	ctx := context.TODO()
	client, err := topologyclientset.NewForConfig(handle.KubeConfig())
	if err != nil {
		klog.ErrorS(err, "Failed to create clientSet for NodeTopologyResource", "kubeConfig", handle.KubeConfig())
		return nil, err
	}

	lister, err := initTopologyInformer(ctx, client)
	if err != nil {
		return nil, err
	}

	topologyMatch := &TopologyMatch{
		PodTopologyCache:       NewPodTopologyCache(ctx, 30*time.Minute),
		handle:                 handle,
		lister:                 lister,
		topologyAwareResources: sets.NewString(cfg.TopologyAwareResources...),
	}

	return topologyMatch, nil
}

func initTopologyInformer(
	ctx context.Context,
	client topologyclientset.Interface,
) (listerv1alpha1.NodeResourceTopologyLister, error) {
	topologyInformerFactory := informers.NewSharedInformerFactory(client, 0)
	nrtLister := topologyInformerFactory.Topology().V1alpha1().NodeResourceTopologies().Lister()

	klog.V(4).InfoS("Start nodeTopologyInformer")
	topologyInformerFactory.Start(ctx.Done())
	topologyInformerFactory.WaitForCacheSync(ctx.Done())
	return nrtLister, nil
}

var _ framework.PreFilterPlugin = &TopologyMatch{}
var _ framework.FilterPlugin = &TopologyMatch{}
var _ framework.ScorePlugin = &TopologyMatch{}
var _ framework.ReservePlugin = &TopologyMatch{}
var _ framework.PreBindPlugin = &TopologyMatch{}

// TopologyMatch plugin which run simplified version of TopologyManager's admit handler
type TopologyMatch struct {
	PodTopologyCache
	handle                 framework.Handle
	lister                 listerv1alpha1.NodeResourceTopologyLister
	topologyAwareResources sets.String
}

// Name returns name of the plugin. It is used in logs, etc.
func (tm *TopologyMatch) Name() string {
	return Name
}

// stateData computed at PreFilter and used at Filter.
type stateData struct {
	sync.Mutex

	aware *bool
	// If not empty, there are containers need to be bound.
	targetContainerIndices  []int
	targetContainerResource *framework.Resource
	// all available NUMA node will be recorded into this map
	podTopologyByNode map[string]*nodeWrapper

	topologyResult topologyv1alpha1.ZoneList
}

// Clone the prefilter stateData.
func (s *stateData) Clone() framework.StateData {
	return s
}

func getStateData(state *framework.CycleState) (*stateData, error) {
	c, err := state.Read(stateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to read %q from cycleState: %w", stateKey, err)
	}

	s, ok := c.(*stateData)
	if !ok {
		return nil, fmt.Errorf("%+v convert to NodeResourcesTopology.stateData error", c)
	}
	return s, nil
}
