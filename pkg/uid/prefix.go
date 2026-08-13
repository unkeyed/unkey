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
	// A portal session carries three distinct values, and they must not share a
	// prefix: `ps_` is the non-secret row handle (safe to log, referenced by
	// audit logs), while `pst_` and `pat_` are bearer credentials. Same-prefix
	// values are how the wrong one ends up in a log line.
	//
	// `pst_` and `pc_` keep their existing values through the portal rename:
	// prefixes are opaque, and they are the one part that would alter issued
	// credentials and stored data rather than code.
	PortalExchangeCodePrefix Prefix = "pst"
	PortalAccessTokenPrefix  Prefix = "pat"
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
