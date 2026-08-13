package uid

// Prefix is a resource type identifier prepended to generated IDs.
type Prefix string

const (
	KeyPrefix                 Prefix = "key"
	APIPrefix                 Prefix = "api"
	RequestPrefix             Prefix = "req"
	WorkspacePrefix           Prefix = "ws"
	KeySpacePrefix            Prefix = "ks" // keyspace
	RolePrefix                Prefix = "role"
	TestPrefix                Prefix = "test" // for tests only
	RatelimitNamespacePrefix  Prefix = "rlns"
	RatelimitOverridePrefix   Prefix = "rlor"
	PermissionPrefix          Prefix = "perm"
	IdentityPrefix            Prefix = "id"
	RatelimitPrefix           Prefix = "rl"
	AuditLogPrefix            Prefix = "log"
	CorrelationPrefix         Prefix = "cor"
	InstancePrefix            Prefix = "ins"
	FrontlinePrefix           Prefix = "fl"
	CiliumNetworkPolicyPrefix Prefix = "net"
	ClusterPrefix             Prefix = "cls"
	RegionPrefix              Prefix = "rgn"
	OrgPrefix                 Prefix = "org"

	// Portal prefixes
	//
	// The wire values are unchanged by the portal rename: prefixes are opaque,
	// and they are the one part that would alter issued credentials and stored
	// data rather than code. Only the Go identifiers follow the new vocabulary.
	PortalExchangeCodePrefix Prefix = "pst"
	PortalSessionPrefix      Prefix = "ps"
	PortalConfigPrefix       Prefix = "pc"

	// Control plane prefixes
	OpenApiSpecPrefix         Prefix = "oas"
	ProjectPrefix             Prefix = "proj"
	EnvironmentPrefix         Prefix = "env"
	EnvironmentVariablePrefix Prefix = "evr"
	AppPrefix                 Prefix = "app"
	DomainPrefix              Prefix = "dom"
	DeploymentPrefix          Prefix = "d"
	FrontlineRoutePrefix      Prefix = "flr"
	CertificatePrefix         Prefix = "cert"
	PolicyPrefix              Prefix = "pol"

	AutoscalingPolicyPrefix Prefix = "asp"
)
