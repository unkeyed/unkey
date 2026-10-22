package privatenetwork

const (
	WorkspaceLabel        = "unkey.com/workspace.id"
	ProjectLabel          = "unkey.com/project.id"
	AppLabel              = "unkey.com/app.id"
	EnvironmentLabel      = "unkey.com/environment.id"
	EnvironmentKindLabel  = "unkey.com/environment.kind"
	DeploymentLabel       = "unkey.com/deployment.id"
	CallerDeploymentLabel = "unkey.com/caller-deployment.id"
	ConnectionLabel       = "unkey.com/connection.id"
	ManagedByLabel        = "app.kubernetes.io/managed-by"
	ComponentLabel        = "app.kubernetes.io/component"

	DiscoveryComponent = "private-dns"
	SourceClusterLabel = "multicluster.kubernetes.io/source-cluster"
	SourceSliceManager = "private-dns.unkey.com"

	CiliumGlobalAnnotation       = "service.cilium.io/global"
	CiliumGlobalSlicesAnnotation = "service.cilium.io/global-sync-endpoint-slices"
)
