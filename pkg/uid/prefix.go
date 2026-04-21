package uid

// Prefix is a resource type identifier prepended to generated IDs.
type Prefix string

const (
	KeyPrefix                  Prefix = "key"
	APIPrefix                  Prefix = "api"
	RequestPrefix              Prefix = "req"
	WorkspacePrefix            Prefix = "ws"
	KeySpacePrefix             Prefix = "ks" // keyspace
	RolePrefix                 Prefix = "role"
	TestPrefix                 Prefix = "test" // for tests only
	RatelimitNamespacePrefix   Prefix = "rlns"
	RatelimitOverridePrefix    Prefix = "rlor"
	PermissionPrefix           Prefix = "perm"
	IdentityPrefix             Prefix = "id"
	RatelimitPrefix            Prefix = "rl"
	AuditLogPrefix             Prefix = "log"
	InstancePrefix             Prefix = "ins"
	SentinelPrefix             Prefix = "s"
	SentinelSubscriptionPrefix Prefix = "sub"
	SentinelTierPrefix         Prefix = "stier"
	CiliumNetworkPolicyPrefix  Prefix = "net"
	ClusterPrefix              Prefix = "cls"
	RegionPrefix               Prefix = "rgn"
	OrgPrefix                  Prefix = "org"

	// Control plane prefixes
	OpenApiSpecPrefix    Prefix = "oas"
	ProjectPrefix        Prefix = "proj"
	EnvironmentPrefix    Prefix = "env"
	AppPrefix            Prefix = "app"
	DomainPrefix         Prefix = "dom"
	DeploymentPrefix     Prefix = "d"
	FrontlineRoutePrefix Prefix = "flr"
	CertificatePrefix    Prefix = "cert"
)
