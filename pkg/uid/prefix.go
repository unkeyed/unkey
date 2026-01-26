package uid

// Prefix is a resource type identifier prepended to generated IDs.
type Prefix string

const (
	KeyPrefix                Prefix = "key"
	PolicyPrefix             Prefix = "pol"
	APIPrefix                Prefix = "api"
	RequestPrefix            Prefix = "req"
	WorkspacePrefix          Prefix = "ws"
	KeySpacePrefix           Prefix = "ks" // keyspace
	VercelBindingPrefix      Prefix = "vb"
	RolePrefix               Prefix = "role"
	TestPrefix               Prefix = "test" // for tests only
	RatelimitNamespacePrefix Prefix = "rlns"
	RatelimitOverridePrefix  Prefix = "rlor"
	PermissionPrefix         Prefix = "perm"
	IdentityPrefix           Prefix = "id"
	RatelimitPrefix          Prefix = "rl"
	AuditLogBucketPrefix     Prefix = "buk"
	AuditLogPrefix           Prefix = "log"
	InstancePrefix           Prefix = "ins"
	SentinelPrefix           Prefix = "se"
	WorkerPrefix             Prefix = "wkr"
	CronJobPrefix            Prefix = "cron"
	KeyEncryptionKeyPrefix   Prefix = "kek"
	OrgPrefix                Prefix = "org"
	WorkflowPrefix           Prefix = "wf"
	StepPrefix               Prefix = "step"

	// Control plane prefixes
	ProjectPrefix        Prefix = "proj"
	EnvironmentPrefix    Prefix = "env"
	VersionPrefix        Prefix = "v"
	BuildPrefix          Prefix = "build"
	RootfsImagePrefix    Prefix = "img"
	DomainPrefix         Prefix = "dom"
	DeploymentPrefix     Prefix = "d"
	FrontlineRoutePrefix Prefix = "ir"
	CertificatePrefix    Prefix = "cert"
	ConnectionPrefix     Prefix = "conn"
)
