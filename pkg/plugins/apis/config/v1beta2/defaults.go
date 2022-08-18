package v1beta2

var (
	defaultNodeResource = []string{"cpu"}
)

func SetDefaults_DynamicArgs(obj *DynamicArgs) {
	if obj.PolicyConfigPath == "" {
		obj.PolicyConfigPath = "/etc/kubernetes/dynamic-scheduler-policy.yaml"
	}
	return
}

func SetDefaults_NodeResourceTopologyMatchArgs(obj *NodeResourceTopologyMatchArgs) {
	if len(obj.TopologyAwareResources) == 0 {
		obj.TopologyAwareResources = defaultNodeResource
	}
	return
}
