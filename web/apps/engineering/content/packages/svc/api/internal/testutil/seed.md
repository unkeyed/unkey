---
title: seed
description: "provides database seeding utilities for integration tests"
---

Package seed provides database seeding utilities for integration tests.

This package handles creating test data in the database with proper relationships between entities. It generates unique IDs, handles foreign key constraints, and provides sensible defaults while allowing full customization.

### Key Types

\[Seeder] is the main type that provides methods to create test entities. It holds a database connection and vault service for encrypting keys. \[Resources] contains the baseline entities created during initial seeding.

### Usage

The seeder is typically used through \[testutil.Harness], which wraps it with context management. For direct usage:

	seeder := seed.New(t, database, vaultService)
	seeder.Seed(ctx)  // Creates baseline data

	api := seeder.CreateAPI(ctx, seed.CreateApiRequest{
	    WorkspaceID: seeder.Resources.UserWorkspace.ID,
	})

	key := seeder.CreateKey(ctx, seed.CreateKeyRequest{
	    WorkspaceID: api.WorkspaceID,
	    KeySpaceID:  api.KeyAuthID.String,
	    Permissions: []seed.CreatePermissionRequest{{Name: "read", Slug: "read", WorkspaceID: api.WorkspaceID}},
	})

### Entity Relationships

The seeder handles cascading entity creation. For example, \[CreateKeyRequest] can include permissions, roles, and rate limits which are created and linked automatically. Similarly, \[CreateRoleRequest] can include permissions to attach.

### Request Types

Each Create\* method has a corresponding request struct that documents all available options. Required fields are typically WorkspaceID and identifiers. Optional fields use pointers to distinguish between "not set" and "set to zero value".

## Types

### type CreateApiRequest

```go
type CreateApiRequest struct {
	WorkspaceID   string
	IpWhitelist   string
	EncryptedKeys bool
	Name          *string
	CreatedAt     *int64
	DefaultPrefix *string
	DefaultBytes  *int32
}
```

CreateApiRequest configures the API to create.

### type CreateDeploymentRequest

```go
type CreateDeploymentRequest struct {
	ID            string
	WorkspaceID   string
	ProjectID     string
	EnvironmentID string
	GitBranch     string
}
```

CreateDeploymentRequest configures the deployment to create.

### type CreateEnvironmentRequest

```go
type CreateEnvironmentRequest struct {
	ID               string
	WorkspaceID      string
	ProjectID        string
	Slug             string
	Description      string
	SentinelConfig   []byte
	DeleteProtection bool
}
```

CreateEnvironmentRequest configures the environment to create.

### type CreateIdentityRequest

```go
type CreateIdentityRequest struct {
	WorkspaceID string
	ExternalID  string
	Meta        []byte
	Ratelimits  []CreateRatelimitRequest
}
```

CreateIdentityRequest configures the identity to create. ExternalID and WorkspaceID are required.

### type CreateKeyRequest

```go
type CreateKeyRequest struct {
	Disabled       bool
	WorkspaceID    string
	KeySpaceID     string
	Remaining      *int32
	IdentityID     *string
	Meta           *string
	Expires        *time.Time
	Name           *string
	Deleted        bool
	ForWorkspaceID *string

	Recoverable bool

	RefillAmount *int32
	RefillDay    *int16

	Permissions []CreatePermissionRequest
	Roles       []CreateRoleRequest
	Ratelimits  []CreateRatelimitRequest
}
```

CreateKeyRequest configures the key to create. WorkspaceID and KeySpaceID are required. The key is enabled by default unless Disabled is true.

### type CreateKeyResponse

```go
type CreateKeyResponse struct {
	KeyID string
	Key   string

	RolesIds      []string
	PermissionIds []string
}
```

CreateKeyResponse contains the created key's ID and raw value, plus the IDs of any roles and permissions that were created and attached.

### type CreatePermissionRequest

```go
type CreatePermissionRequest struct {
	Name        string
	Slug        string
	Description *string
	WorkspaceID string
}
```

CreatePermissionRequest configures the permission to create. Name, Slug, and WorkspaceID are required.

### type CreateProjectRequest

```go
type CreateProjectRequest struct {
	ID               string
	WorkspaceID      string
	Name             string
	Slug             string
	GitRepositoryURL string
	DefaultBranch    string
	DeleteProtection bool
}
```

CreateProjectRequest configures the project to create.

### type CreateRatelimitRequest

```go
type CreateRatelimitRequest struct {
	Name        string
	WorkspaceID string
	AutoApply   bool
	Duration    int64
	Limit       int32
	IdentityID  *string
	KeyID       *string
}
```

CreateRatelimitRequest configures the rate limit to create. Either IdentityID or KeyID must be set to attach the rate limit to an entity.

### type CreateRoleRequest

```go
type CreateRoleRequest struct {
	Name        string
	Description *string
	WorkspaceID string

	Permissions []CreatePermissionRequest
}
```

CreateRoleRequest configures the role to create. Name and WorkspaceID are required.

### type Resources

```go
type Resources struct {
	RootWorkspace db.Workspace
	RootKeySpace  db.KeyAuth
	RootApi       db.Api
	UserWorkspace db.Workspace
}
```

Resources contains the baseline entities created during \[Seeder.Seed]. These represent the "system" workspace used for root keys and a user workspace for test-specific data.

### type Seeder

```go
type Seeder struct {
	t         *testing.T
	DB        db.Database
	Vault     *vault.Service
	Resources Resources
}
```

Seeder provides methods to create test entities in the database. It ensures proper foreign key relationships and generates unique IDs for all entities.

#### func New

```go
func New(t *testing.T, database db.Database, vault *vault.Service) *Seeder
```

New creates a Seeder with the given database and vault service. Call \[Seeder.Seed] after creation to populate baseline data.

#### func (Seeder) CreateAPI

```go
func (s *Seeder) CreateAPI(ctx context.Context, req CreateApiRequest) db.Api
```

CreateAPI creates an API and its associated key space. The key space is created first since the API references it. Returns the created API which includes the KeyAuthID linking to the key space.

#### func (Seeder) CreateDeployment

```go
func (s *Seeder) CreateDeployment(ctx context.Context, req CreateDeploymentRequest) db.Deployment
```

CreateDeployment creates a deployment within a project and environment.

#### func (Seeder) CreateEnvironment

```go
func (s *Seeder) CreateEnvironment(ctx context.Context, req CreateEnvironmentRequest) db.Environment
```

CreateEnvironment creates an environment within a project. If SentinelConfig is nil or empty, it defaults to "{}".

#### func (Seeder) CreateIdentity

```go
func (s *Seeder) CreateIdentity(ctx context.Context, req CreateIdentityRequest) db.Identity
```

CreateIdentity creates an identity with optional rate limits attached. If Meta is nil or empty, it defaults to "{}". Any rate limits in Ratelimits are created and linked to this identity.

#### func (Seeder) CreateKey

```go
func (s *Seeder) CreateKey(ctx context.Context, req CreateKeyRequest) CreateKeyResponse
```

CreateKey creates a key with the specified configuration. If Permissions, Roles, or Ratelimits are provided, they are created and linked to the key. If Deleted is true, the key is soft-deleted after creation. If Recoverable is true and a Vault service is configured, the key is encrypted and stored for recovery.

#### func (Seeder) CreatePermission

```go
func (s *Seeder) CreatePermission(ctx context.Context, req CreatePermissionRequest) db.Permission
```

CreatePermission creates a permission that can be attached to keys or roles.

#### func (Seeder) CreateProject

```go
func (h *Seeder) CreateProject(ctx context.Context, req CreateProjectRequest) db.Project
```

CreateProject creates a project within a workspace. The ID should be generated with \[uid.New] using \[uid.ProjectPrefix].

#### func (Seeder) CreateRatelimit

```go
func (s *Seeder) CreateRatelimit(ctx context.Context, req CreateRatelimitRequest) db.Ratelimit
```

CreateRatelimit creates a rate limit attached to either a key or identity. The rate limit allows Limit requests per Duration (in milliseconds). If AutoApply is true, the rate limit is automatically applied during key verification.

#### func (Seeder) CreateRole

```go
func (s *Seeder) CreateRole(ctx context.Context, req CreateRoleRequest) db.Role
```

CreateRole creates a role with optional permissions attached. Any permissions in Permissions are created and linked to this role.

#### func (Seeder) CreateRootKey

```go
func (s *Seeder) CreateRootKey(ctx context.Context, workspaceID string, permissions ...string) string
```

CreateRootKey creates a root key that authorizes operations on the specified workspace. The key is created in the root key space (from baseline seed data). Pass permission names to grant; if a permission already exists, it reuses the existing one. Returns the raw key value for use in Authorization headers.

#### func (Seeder) CreateWorkspace

```go
func (s *Seeder) CreateWorkspace(ctx context.Context) db.Workspace
```

CreateWorkspace creates a new workspace with auto-generated IDs for the workspace, org, name, and slug.

#### func (Seeder) Seed

```go
func (s *Seeder) Seed(ctx context.Context)
```

Seed initializes the database with baseline test data. This creates a root workspace (for issuing root keys), a root API with its key space, and a user workspace for test-specific entities. The created resources are stored in \[Seeder.Resources].

