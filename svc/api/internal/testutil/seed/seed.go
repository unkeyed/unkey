package seed

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/db"
	dbtype "github.com/unkeyed/unkey/pkg/db/types"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/vault"
)

// Resources contains the baseline entities created during [Seeder.Seed]. These
// represent the "system" workspace used for root keys and a user workspace for
// test-specific data.
type Resources struct {
	RootWorkspace db.Workspace
	RootKeySpace  db.KeyAuth
	RootApi       db.Api
	UserWorkspace db.Workspace
}

// Seeder provides methods to create test entities in the database. It ensures proper
// foreign key relationships and generates unique IDs for all entities.
type Seeder struct {
	t         *testing.T
	DB        db.Database
	Vault     *vault.Service
	Resources Resources
}

// New creates a Seeder with the given database and vault service. Call [Seeder.Seed]
// after creation to populate baseline data.
func New(t *testing.T, database db.Database, vault *vault.Service) *Seeder {
	return &Seeder{
		t:         t,
		DB:        database,
		Vault:     vault,
		Resources: Resources{}, //nolint:exhaustruct
	}
}

// CreateWorkspace creates a new workspace with auto-generated IDs for the workspace,
// org, name, and slug.
func (s *Seeder) CreateWorkspace(ctx context.Context) db.Workspace {
	params := db.InsertWorkspaceParams{
		ID:        uid.New("test_ws"),
		OrgID:     uid.New("test_org"),
		Name:      uid.New("test_name"),
		Slug:      uid.New("slug"),
		CreatedAt: time.Now().UnixMilli(),
	}

	err := db.Query.InsertWorkspace(ctx, s.DB.RW(), params)
	require.NoError(s.t, err)

	ws, err := db.Query.FindWorkspaceByID(ctx, s.DB.RW(), params.ID)
	require.NoError(s.t, err)

	return ws
}

// Seed initializes the database with baseline test data. This creates a root workspace
// (for issuing root keys), a root API with its key space, and a user workspace for
// test-specific entities. The created resources are stored in [Seeder.Resources].
func (s *Seeder) Seed(ctx context.Context) {
	s.Resources.UserWorkspace = s.CreateWorkspace(ctx)
	s.Resources.RootWorkspace = s.CreateWorkspace(ctx)
	s.Resources.RootApi = s.CreateAPI(ctx, CreateApiRequest{
		WorkspaceID:   s.Resources.RootWorkspace.ID,
		IpWhitelist:   "",
		EncryptedKeys: false,
		Name:          nil,
		CreatedAt:     nil,
		DefaultPrefix: nil,
		DefaultBytes:  nil,
	})
	keySpace, err := db.Query.FindKeySpaceByID(ctx, s.DB.RW(), s.Resources.RootApi.KeyAuthID.String)
	require.NoError(s.t, err)
	s.Resources.RootKeySpace = keySpace
}

// CreateApiRequest configures the API to create.
type CreateApiRequest struct {
	WorkspaceID   string
	IpWhitelist   string
	EncryptedKeys bool
	Name          *string
	CreatedAt     *int64
	DefaultPrefix *string
	DefaultBytes  *int32
}

// CreateAPI creates an API and its associated key space. The key space is created
// first since the API references it. Returns the created API which includes the
// KeyAuthID linking to the key space.
func (s *Seeder) CreateAPI(ctx context.Context, req CreateApiRequest) db.Api {
	keySpaceID := uid.New(uid.KeySpacePrefix)
	err := db.Query.InsertKeySpace(ctx, s.DB.RW(), db.InsertKeySpaceParams{
		ID:                 keySpaceID,
		WorkspaceID:        req.WorkspaceID,
		CreatedAtM:         time.Now().UnixMilli(),
		DefaultPrefix:      sql.NullString{String: ptr.SafeDeref(req.DefaultPrefix), Valid: req.DefaultPrefix != nil},
		DefaultBytes:       sql.NullInt32{Int32: ptr.SafeDeref(req.DefaultBytes), Valid: req.DefaultBytes != nil},
		StoreEncryptedKeys: req.EncryptedKeys,
	})
	require.NoError(s.t, err)

	apiID := uid.New("api")
	err = db.Query.InsertApi(ctx, s.DB.RW(), db.InsertApiParams{
		ID:          apiID,
		Name:        ptr.SafeDeref(req.Name, "test-api"),
		WorkspaceID: req.WorkspaceID,
		IpWhitelist: sql.NullString{String: req.IpWhitelist, Valid: req.IpWhitelist != ""},
		AuthType:    db.NullApisAuthType{Valid: true, ApisAuthType: db.ApisAuthTypeKey},
		KeyAuthID:   sql.NullString{Valid: true, String: keySpaceID},
		CreatedAtM:  ptr.SafeDeref(req.CreatedAt, time.Now().UnixMilli()),
	})
	require.NoError(s.t, err)

	api, err := db.Query.FindApiByID(ctx, s.DB.RW(), apiID)
	require.NoError(s.t, err)

	return api
}

// CreateProjectRequest configures the project to create.
type CreateProjectRequest struct {
	ID               string
	WorkspaceID      string
	Name             string
	Slug             string
	DefaultBranch    string
	DeleteProtection bool
}

// CreateProject creates a project within a workspace. The ID should be generated
// with [uid.New] using [uid.ProjectPrefix].
func (h *Seeder) CreateProject(ctx context.Context, req CreateProjectRequest) db.Project {
	err := db.Query.InsertProject(ctx, h.DB.RW(), db.InsertProjectParams{
		ID:               req.ID,
		WorkspaceID:      req.WorkspaceID,
		Name:             req.Name,
		Slug:             req.Slug,
		DefaultBranch:    sql.NullString{Valid: true, String: req.DefaultBranch},
		DeleteProtection: sql.NullBool{Valid: true, Bool: req.DeleteProtection},
		CreatedAt:        time.Now().UnixMilli(),
		UpdatedAt:        sql.NullInt64{Int64: 0, Valid: false},
	})
	require.NoError(h.t, err)

	project, err := db.Query.FindProjectById(ctx, h.DB.RO(), req.ID)
	require.NoError(h.t, err)

	return db.Project{
		ID:               project.ID,
		WorkspaceID:      project.WorkspaceID,
		Name:             project.Name,
		Slug:             project.Slug,
		DefaultBranch:    project.DefaultBranch,
		DeleteProtection: project.DeleteProtection,
		CreatedAt:        project.CreatedAt,
		UpdatedAt:        project.UpdatedAt,
		Pk:               0,
		LiveDeploymentID: sql.NullString{String: "", Valid: false},
		IsRolledBack:     false,
		DepotProjectID:   sql.NullString{String: "", Valid: false},
	}
}

// CreateEnvironmentRequest configures the environment to create.
type CreateEnvironmentRequest struct {
	ID               string
	WorkspaceID      string
	ProjectID        string
	Slug             string
	Description      string
	SentinelConfig   []byte
	DeleteProtection bool
}

// CreateEnvironment creates an environment within a project. If SentinelConfig is
// nil or empty, it defaults to "{}".
func (s *Seeder) CreateEnvironment(ctx context.Context, req CreateEnvironmentRequest) db.Environment {
	sentinelConfig := []byte("{}")
	if len(req.SentinelConfig) > 0 {
		sentinelConfig = req.SentinelConfig
	}

	err := db.Query.InsertEnvironment(ctx, s.DB.RW(), db.InsertEnvironmentParams{
		ID:             req.ID,
		WorkspaceID:    req.WorkspaceID,
		ProjectID:      req.ProjectID,
		Slug:           req.Slug,
		Description:    req.Description,
		SentinelConfig: sentinelConfig,
		CreatedAt:      time.Now().UnixMilli(),
		UpdatedAt:      sql.NullInt64{Int64: 0, Valid: false},
	})
	require.NoError(s.t, err)

	environment, err := db.Query.FindEnvironmentById(ctx, s.DB.RO(), req.ID)
	require.NoError(s.t, err)

	return db.Environment{
		Pk:               0,
		ID:               environment.ID,
		WorkspaceID:      environment.WorkspaceID,
		ProjectID:        environment.ProjectID,
		Slug:             environment.Slug,
		Description:      req.Description,
		SentinelConfig:   sentinelConfig,
		DeleteProtection: sql.NullBool{Valid: true, Bool: req.DeleteProtection},
		CreatedAt:        time.Now().UnixMilli(),
		UpdatedAt:        sql.NullInt64{Int64: 0, Valid: false},
	}
}

// CreateRootKey creates a root key that authorizes operations on the specified
// workspace. The key is created in the root key space (from baseline seed data).
// Pass permission names to grant; if a permission already exists, it reuses the
// existing one. Returns the raw key value for use in Authorization headers.
func (s *Seeder) CreateRootKey(ctx context.Context, workspaceID string, permissions ...string) string {
	key := uid.New("test_root_key")

	insertKeyParams := db.InsertKeyParams{
		ID:                 uid.New("test_root_key"),
		Hash:               hash.Sha256(key),
		WorkspaceID:        s.Resources.RootWorkspace.ID,
		ForWorkspaceID:     sql.NullString{String: workspaceID, Valid: true},
		KeySpaceID:         s.Resources.RootKeySpace.ID,
		Start:              key[:4],
		CreatedAtM:         time.Now().UnixMilli(),
		Enabled:            true,
		Name:               sql.NullString{String: "", Valid: false},
		IdentityID:         sql.NullString{String: "", Valid: false},
		Meta:               sql.NullString{String: "", Valid: false},
		Expires:            sql.NullTime{Time: time.Time{}, Valid: false},
		RemainingRequests:  sql.NullInt32{Int32: 0, Valid: false},
		RefillDay:          sql.NullInt16{Int16: 0, Valid: false},
		RefillAmount:       sql.NullInt32{Int32: 0, Valid: false},
		PendingMigrationID: sql.NullString{Valid: false, String: ""},
	}

	err := db.Query.InsertKey(ctx, s.DB.RW(), insertKeyParams)
	require.NoError(s.t, err)

	if len(permissions) > 0 {
		for _, permission := range permissions {
			permissionID := uid.New(uid.TestPrefix)
			err := db.Query.InsertPermission(ctx, s.DB.RW(), db.InsertPermissionParams{
				PermissionID: permissionID,
				WorkspaceID:  s.Resources.RootWorkspace.ID,
				Name:         permission,
				Slug:         permission,
				Description:  dbtype.NullString{String: "", Valid: false},
				CreatedAtM:   time.Now().UnixMilli(),
			})

			mysqlErr := &mysql.MySQLError{} // nolint:exhaustruct
			if errors.As(err, &mysqlErr) {
				require.True(s.t, db.IsDuplicateKeyError(err), "Expected duplicate key error, got MySQL error number %d", mysqlErr.Number)
				existing, findErr := db.Query.FindPermissionByNameAndWorkspaceID(ctx, s.DB.RO(), db.FindPermissionByNameAndWorkspaceIDParams{
					WorkspaceID: s.Resources.RootWorkspace.ID,
					Name:        permission,
				})
				require.NoError(s.t, findErr)
				permissionID = existing.ID

			} else {
				require.NoError(s.t, err)
			}

			err = db.Query.InsertKeyPermission(ctx, s.DB.RW(), db.InsertKeyPermissionParams{
				PermissionID: permissionID,
				KeyID:        insertKeyParams.ID,
				WorkspaceID:  s.Resources.RootWorkspace.ID,
				CreatedAt:    time.Now().UnixMilli(),
				UpdatedAt:    sql.NullInt64{Int64: 0, Valid: false},
			})
			require.NoError(s.t, err)
		}
	}

	return key
}

// CreateKeyRequest configures the key to create. WorkspaceID and KeySpaceID are
// required. The key is enabled by default unless Disabled is true.
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

// CreateKeyResponse contains the created key's ID and raw value, plus the IDs of
// any roles and permissions that were created and attached.
type CreateKeyResponse struct {
	KeyID string
	Key   string

	RolesIds      []string
	PermissionIds []string
}

// CreateKey creates a key with the specified configuration. If Permissions, Roles,
// or Ratelimits are provided, they are created and linked to the key. If Deleted
// is true, the key is soft-deleted after creation. If Recoverable is true and a
// Vault service is configured, the key is encrypted and stored for recovery.
func (s *Seeder) CreateKey(ctx context.Context, req CreateKeyRequest) CreateKeyResponse {
	keyID := uid.New(uid.KeyPrefix)
	key := uid.New("")
	start := key[:4]

	err := db.Query.InsertKey(ctx, s.DB.RW(), db.InsertKeyParams{
		ID:                 keyID,
		KeySpaceID:         req.KeySpaceID,
		WorkspaceID:        req.WorkspaceID,
		CreatedAtM:         time.Now().UnixMilli(),
		Hash:               hash.Sha256(key),
		Enabled:            !req.Disabled,
		Start:              start,
		Name:               sql.NullString{String: ptr.SafeDeref(req.Name, "test-key"), Valid: true},
		ForWorkspaceID:     sql.NullString{String: ptr.SafeDeref(req.ForWorkspaceID, ""), Valid: req.ForWorkspaceID != nil},
		Meta:               sql.NullString{String: ptr.SafeDeref(req.Meta, ""), Valid: req.Meta != nil},
		IdentityID:         sql.NullString{String: ptr.SafeDeref(req.IdentityID, ""), Valid: req.IdentityID != nil},
		Expires:            sql.NullTime{Time: ptr.SafeDeref(req.Expires, time.Time{}), Valid: req.Expires != nil},
		RemainingRequests:  sql.NullInt32{Int32: ptr.SafeDeref(req.Remaining, 0), Valid: req.Remaining != nil},
		RefillAmount:       sql.NullInt32{Int32: ptr.SafeDeref(req.RefillAmount, 0), Valid: req.RefillAmount != nil},
		RefillDay:          sql.NullInt16{Int16: ptr.SafeDeref(req.RefillDay, 0), Valid: req.RefillDay != nil},
		PendingMigrationID: sql.NullString{Valid: false, String: ""},
	})
	require.NoError(s.t, err)

	res := CreateKeyResponse{
		KeyID:         keyID,
		Key:           key,
		RolesIds:      []string{},
		PermissionIds: []string{},
	}

	if req.Deleted {
		err = db.Query.SoftDeleteKeyByID(ctx, s.DB.RW(), db.SoftDeleteKeyByIDParams{
			Now: sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
			ID:  keyID,
		})

		require.NoError(s.t, err)
	}

	if req.Recoverable && s.Vault != nil {
		var encryption *vaultv1.EncryptResponse
		encryption, err = s.Vault.Encrypt(ctx, &vaultv1.EncryptRequest{
			Keyring: req.WorkspaceID,
			Data:    key,
		})
		require.NoError(s.t, err)
		err = db.Query.InsertKeyEncryption(ctx, s.DB.RW(), db.InsertKeyEncryptionParams{
			WorkspaceID:     req.WorkspaceID,
			KeyID:           keyID,
			CreatedAt:       time.Now().UnixMilli(),
			Encrypted:       encryption.GetEncrypted(),
			EncryptionKeyID: encryption.GetKeyId(),
		})
		require.NoError(s.t, err)
	}

	for _, role := range req.Roles {
		r := s.CreateRole(ctx, role)
		err = db.Query.InsertKeyRole(ctx, s.DB.RW(), db.InsertKeyRoleParams{
			KeyID:       keyID,
			RoleID:      r.ID,
			WorkspaceID: req.WorkspaceID,
			CreatedAtM:  time.Now().UnixMilli(),
		})
		require.NoError(s.t, err)
		res.RolesIds = append(res.RolesIds, r.ID)
	}

	for _, permission := range req.Permissions {
		perm := s.CreatePermission(ctx, permission)
		err = db.Query.InsertKeyPermission(ctx, s.DB.RW(), db.InsertKeyPermissionParams{
			KeyID:        keyID,
			PermissionID: perm.ID,
			WorkspaceID:  req.WorkspaceID,
			CreatedAt:    time.Now().UnixMilli(),
			UpdatedAt:    sql.NullInt64{Int64: 0, Valid: false},
		})

		require.NoError(s.t, err)
		res.PermissionIds = append(res.PermissionIds, perm.ID)
	}

	for _, ratelimit := range req.Ratelimits {
		ratelimit.KeyID = ptr.P(keyID)
		s.CreateRatelimit(ctx, ratelimit)
	}

	return res
}

// CreateRatelimitRequest configures the rate limit to create. Either IdentityID or
// KeyID must be set to attach the rate limit to an entity.
type CreateRatelimitRequest struct {
	Name        string
	WorkspaceID string
	AutoApply   bool
	Duration    int64
	Limit       int32
	IdentityID  *string
	KeyID       *string
}

// CreateRatelimit creates a rate limit attached to either a key or identity. The
// rate limit allows Limit requests per Duration (in milliseconds). If AutoApply is
// true, the rate limit is automatically applied during key verification.
func (s *Seeder) CreateRatelimit(ctx context.Context, req CreateRatelimitRequest) db.Ratelimit {
	ratelimitID := uid.New(uid.RatelimitPrefix)
	createdAt := time.Now().UnixMilli()
	var err error

	if req.IdentityID != nil {
		err = db.Query.InsertIdentityRatelimit(ctx, s.DB.RW(), db.InsertIdentityRatelimitParams{
			ID:          ratelimitID,
			WorkspaceID: req.WorkspaceID,
			IdentityID:  sql.NullString{String: *req.IdentityID, Valid: true},
			Name:        req.Name,
			Limit:       req.Limit,
			Duration:    req.Duration,
			AutoApply:   req.AutoApply,
			CreatedAt:   createdAt,
		})
	}

	if req.KeyID != nil {
		err = db.Query.InsertKeyRatelimit(ctx, s.DB.RW(), db.InsertKeyRatelimitParams{
			ID:          ratelimitID,
			WorkspaceID: req.WorkspaceID,
			KeyID:       sql.NullString{String: *req.KeyID, Valid: true},
			Name:        req.Name,
			Limit:       req.Limit,
			Duration:    req.Duration,
			AutoApply:   req.AutoApply,
			UpdatedAt:   sql.NullInt64{Int64: 0, Valid: false},
			CreatedAt:   createdAt,
		})
	}

	require.NoError(s.t, err)

	return db.Ratelimit{
		Pk:          0, // db internal
		ID:          ratelimitID,
		Name:        req.Name,
		WorkspaceID: req.WorkspaceID,
		CreatedAt:   createdAt,
		UpdatedAt:   sql.NullInt64{Valid: false, Int64: 0},
		KeyID:       sql.NullString{String: ptr.SafeDeref(req.KeyID, ""), Valid: req.KeyID != nil},
		IdentityID:  sql.NullString{String: ptr.SafeDeref(req.IdentityID, ""), Valid: req.IdentityID != nil},
		Limit:       req.Limit,
		Duration:    req.Duration,
		AutoApply:   req.AutoApply,
	}
}

// CreateIdentityRequest configures the identity to create. ExternalID and
// WorkspaceID are required.
type CreateIdentityRequest struct {
	WorkspaceID string
	ExternalID  string
	Meta        []byte
	Ratelimits  []CreateRatelimitRequest
}

// CreateIdentity creates an identity with optional rate limits attached. If Meta
// is nil or empty, it defaults to "{}". Any rate limits in Ratelimits are created
// and linked to this identity.
func (s *Seeder) CreateIdentity(ctx context.Context, req CreateIdentityRequest) db.Identity {
	metaBytes := []byte("{}")
	if len(req.Meta) > 0 {
		metaBytes = req.Meta
	}

	require.NoError(s.t, assert.NotEmpty(req.ExternalID, "Identity ExternalID must be set"))
	require.NoError(s.t, assert.NotEmpty(req.WorkspaceID, "Identity WorkspaceID must be set"))

	identityID := uid.New(uid.IdentityPrefix)
	err := db.Query.InsertIdentity(ctx, s.DB.RW(), db.InsertIdentityParams{
		ID:          identityID,
		ExternalID:  req.ExternalID,
		WorkspaceID: req.WorkspaceID,
		Environment: "",
		CreatedAt:   time.Now().UnixMilli(),
		Meta:        metaBytes,
	})
	require.NoError(s.t, err)

	for _, ratelimit := range req.Ratelimits {
		ratelimit.IdentityID = ptr.P(identityID)
		s.CreateRatelimit(ctx, ratelimit)
	}

	return db.Identity{
		Pk:          0, // db internal
		ID:          identityID,
		ExternalID:  req.ExternalID,
		WorkspaceID: req.WorkspaceID,
		Environment: "",
		Meta:        metaBytes,
		Deleted:     false,
		CreatedAt:   time.Now().UnixMilli(),
		UpdatedAt:   sql.NullInt64{Valid: false, Int64: 0},
	}
}

// CreateRoleRequest configures the role to create. Name and WorkspaceID are required.
type CreateRoleRequest struct {
	Name        string
	Description *string
	WorkspaceID string

	Permissions []CreatePermissionRequest
}

// CreateRole creates a role with optional permissions attached. Any permissions in
// Permissions are created and linked to this role.
func (s *Seeder) CreateRole(ctx context.Context, req CreateRoleRequest) db.Role {
	require.NoError(s.t, assert.NotEmpty(req.WorkspaceID, "Role WorkspaceID must be set"))
	require.NoError(s.t, assert.NotEmpty(req.Name, "Role Name must be set"))

	roleID := uid.New(uid.RolePrefix)
	createdAt := time.Now().UnixMilli()

	err := db.Query.InsertRole(ctx, s.DB.RW(), db.InsertRoleParams{
		RoleID:      roleID,
		WorkspaceID: req.WorkspaceID,
		Name:        req.Name,
		CreatedAt:   createdAt,
		Description: sql.NullString{Valid: req.Description != nil, String: ptr.SafeDeref(req.Description, "")},
	})
	require.NoError(s.t, err)

	for _, permission := range req.Permissions {
		perm := s.CreatePermission(ctx, permission)
		err = db.Query.InsertRolePermission(ctx, s.DB.RW(), db.InsertRolePermissionParams{
			RoleID:       roleID,
			PermissionID: perm.ID,
			WorkspaceID:  req.WorkspaceID,
			CreatedAtM:   time.Now().UnixMilli(),
		})
		require.NoError(s.t, err)
	}

	return db.Role{
		Pk:          0, // db internal
		ID:          roleID,
		WorkspaceID: req.WorkspaceID,
		Name:        req.Name,
		Description: sql.NullString{Valid: req.Description != nil, String: ptr.SafeDeref(req.Description, "")},
		CreatedAtM:  createdAt,
		UpdatedAtM:  sql.NullInt64{Valid: false, Int64: 0},
	}
}

// CreatePermissionRequest configures the permission to create. Name, Slug, and
// WorkspaceID are required.
type CreatePermissionRequest struct {
	Name        string
	Slug        string
	Description *string
	WorkspaceID string
}

// CreateDeploymentRequest configures the deployment to create.
type CreateDeploymentRequest struct {
	ID            string
	WorkspaceID   string
	ProjectID     string
	EnvironmentID string
	GitBranch     string
}

// CreateDeployment creates a deployment within a project and environment.
func (s *Seeder) CreateDeployment(ctx context.Context, req CreateDeploymentRequest) db.Deployment {
	require.NoError(s.t, assert.NotEmpty(req.ID, "Deployment ID must be set"))
	require.NoError(s.t, assert.NotEmpty(req.WorkspaceID, "Deployment WorkspaceID must be set"))
	require.NoError(s.t, assert.NotEmpty(req.ProjectID, "Deployment ProjectID must be set"))
	require.NoError(s.t, assert.NotEmpty(req.EnvironmentID, "Deployment EnvironmentID must be set"))

	createdAt := time.Now().UnixMilli()
	err := db.Query.InsertDeployment(ctx, s.DB.RW(), db.InsertDeploymentParams{
		ID:                            req.ID,
		K8sName:                       "test-" + req.ID,
		WorkspaceID:                   req.WorkspaceID,
		ProjectID:                     req.ProjectID,
		EnvironmentID:                 req.EnvironmentID,
		GitCommitSha:                  sql.NullString{Valid: false},
		GitBranch:                     sql.NullString{String: req.GitBranch, Valid: req.GitBranch != ""},
		SentinelConfig:                []byte("{}"),
		GitCommitMessage:              sql.NullString{Valid: false},
		GitCommitAuthorHandle:         sql.NullString{Valid: false},
		GitCommitAuthorAvatarUrl:      sql.NullString{Valid: false},
		GitCommitTimestamp:            sql.NullInt64{Valid: false},
		OpenapiSpec:                   sql.NullString{Valid: false},
		EncryptedEnvironmentVariables: []byte{},
		Command:                       nil,
		Status:                        db.DeploymentsStatusPending,
		CpuMillicores:                 100,
		MemoryMib:                     128,
		Port:                          8080,
		RestartPolicy:                 db.DeploymentsRestartPolicyAlways,
		ShutdownSignal:                db.DeploymentsShutdownSignalSIGTERM,
		Healthcheck:                   dbtype.NullHealthcheck{},
		CreatedAt:                     createdAt,
		UpdatedAt:                     sql.NullInt64{Valid: false},
	})
	require.NoError(s.t, err)

	deployment, err := db.Query.FindDeploymentById(ctx, s.DB.RO(), req.ID)
	require.NoError(s.t, err)

	return deployment
}

// CreatePermission creates a permission that can be attached to keys or roles.
func (s *Seeder) CreatePermission(ctx context.Context, req CreatePermissionRequest) db.Permission {
	require.NoError(s.t, assert.NotEmpty(req.WorkspaceID, "Permission WorkspaceID must be set"))
	require.NoError(s.t, assert.NotEmpty(req.Name, "Permission Name must be set"))
	require.NoError(s.t, assert.NotEmpty(req.Slug, "Permission Slug must be set"))

	permissionID := uid.New(uid.PermissionPrefix)
	createdAt := time.Now().UnixMilli()

	err := db.Query.InsertPermission(ctx, s.DB.RW(), db.InsertPermissionParams{
		PermissionID: permissionID,
		WorkspaceID:  req.WorkspaceID,
		Name:         req.Name,
		Slug:         req.Slug,
		Description:  dbtype.NullString{Valid: req.Description != nil, String: ptr.SafeDeref(req.Description, "")},
		CreatedAtM:   createdAt,
	})
	require.NoError(s.t, err)

	return db.Permission{
		Pk:          0, // db internal
		ID:          permissionID,
		WorkspaceID: req.WorkspaceID,
		Name:        req.Name,
		Slug:        req.Slug,
		Description: dbtype.NullString{Valid: req.Description != nil, String: ptr.SafeDeref(req.Description, "")},
		CreatedAtM:  createdAt,
		UpdatedAtM:  sql.NullInt64{Valid: false, Int64: 0},
	}
}
