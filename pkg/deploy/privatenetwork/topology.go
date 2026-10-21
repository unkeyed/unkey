package privatenetwork

const (
	// TopologyConfigMap identifies the cluster-region catalog in kube-system.
	TopologyConfigMap = "unkey-private-network-topology"
	// TopologyComponent separates topology from per-connection discovery objects.
	TopologyComponent = "private-dns-topology"
	// MaxRegionLength bounds a region DNS label so that
	// [RegionalConnectionServiceName] fits in a Service name.
	MaxRegionLength = 22
)
