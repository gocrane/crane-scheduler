package v1beta3

var (
	defaultNodeResource = []string{"cpu"}
)

func SetDefaults_DynamicArgs(obj *DynamicArgs) {
	if obj.PolicyConfigPath == nil {
		path := "/etc/kubernetes/dynamic-scheduler-policy.yaml"
		obj.PolicyConfigPath = &path
	}
	return
}

func SetDefaults_NodeResourceTopologyMatchArgs(obj *NodeResourceTopologyMatchArgs) {
	if len(obj.TopologyAwareResources) == 0 {
		obj.TopologyAwareResources = defaultNodeResource
	}
	return
}
