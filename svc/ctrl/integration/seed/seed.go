package seed

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/gen/proto/vault/v1/vaultv1connect"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/db"
	dbtype "github.com/unkeyed/unkey/pkg/db/types"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/uid"
)

// Resources represents seed data created for tests
type Resources struct {
	RootWorkspace db.Workspace
	RootKeySpace  db.KeyAuth
	RootApi       db.Api
	UserWorkspace db.Workspace
}

// Seeder provides methods to seed test data
type Seeder struct {
	t         *testing.T
	DB        db.Database
	Vault     vaultv1connect.VaultServiceClient
	Resources Resources
}

// New creates a new Seeder instance
func New(t *testing.T, database db.Database, vault vaultv1connect.VaultServiceClient) *Seeder {
	return &Seeder{
		t:         t,
		DB:        database,
		Vault:     vault,
		Resources: Resources{}, //nolint:exhaustruct
	}
}

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

// Seed initializes the database with test data
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

type CreateApiRequest struct {
	WorkspaceID   string
	IpWhitelist   string
	EncryptedKeys bool
	Name          *string
	CreatedAt     *int64
	DefaultPrefix *string
	DefaultBytes  *int32
}

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

type CreateProjectRequest struct {
	ID               string
	WorkspaceID      string
	Name             string
	Slug             string
	DefaultBranch    string
	DeleteProtection bool
}

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

type CreateEnvironmentRequest struct {
	ID               string
	WorkspaceID      string
	ProjectID        string
	Slug             string
	Description      string
	SentinelConfig   []byte
	DeleteProtection bool
}

func (s *Seeder) CreateEnvironment(ctx context.Context, req CreateEnvironmentRequest) db.Environment {
	sentinelConfig := []byte("{}")
	if len(req.SentinelConfig) > 0 {
		sentinelConfig = req.SentinelConfig
	}

	now := time.Now().UnixMilli()

	err := db.Query.InsertEnvironment(ctx, s.DB.RW(), db.InsertEnvironmentParams{
		ID:             req.ID,
		WorkspaceID:    req.WorkspaceID,
		ProjectID:      req.ProjectID,
		Slug:           req.Slug,
		Description:    req.Description,
		SentinelConfig: sentinelConfig,
		CreatedAt:      now,
		UpdatedAt:      sql.NullInt64{Int64: 0, Valid: false},
	})
	require.NoError(s.t, err)

	err = db.Query.UpsertEnvironmentRuntimeSettings(ctx, s.DB.RW(), db.UpsertEnvironmentRuntimeSettingsParams{
		WorkspaceID:    req.WorkspaceID,
		EnvironmentID:  req.ID,
		Port:           8080,
		CpuMillicores:  256,
		MemoryMib:      256,
		Command:        dbtype.StringSlice{},
		Healthcheck:    dbtype.NullHealthcheck{Healthcheck: nil, Valid: false},
		RegionConfig:   dbtype.RegionConfig{},
		ShutdownSignal: db.EnvironmentRuntimeSettingsShutdownSignalSIGTERM,
		CreatedAt:      now,
		UpdatedAt:      sql.NullInt64{Valid: true, Int64: now},
	})
	require.NoError(s.t, err)

	err = db.Query.UpsertEnvironmentBuildSettings(ctx, s.DB.RW(), db.UpsertEnvironmentBuildSettingsParams{
		WorkspaceID:   req.WorkspaceID,
		EnvironmentID: req.ID,
		Dockerfile:    "Dockerfile",
		DockerContext: ".",
		CreatedAt:     now,
		UpdatedAt:     sql.NullInt64{Valid: true, Int64: now},
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
		CreatedAt:        now,
		UpdatedAt:        sql.NullInt64{Int64: 0, Valid: false},
	}
}

type CreateDeploymentRequest struct {
	ID            string
	WorkspaceID   string
	ProjectID     string
	EnvironmentID string
	Status        db.DeploymentsStatus
	CreatedAt     int64
	UpdatedAt     sql.NullInt64
}

func (s *Seeder) CreateDeployment(ctx context.Context, req CreateDeploymentRequest) db.Deployment {
	id := req.ID
	if id == "" {
		id = uid.New(uid.DeploymentPrefix)
	}

	createdAt := req.CreatedAt
	if createdAt == 0 {
		createdAt = time.Now().UnixMilli()
	}

	err := db.Query.InsertDeployment(ctx, s.DB.RW(), db.InsertDeploymentParams{
		ID:                            id,
		K8sName:                       uid.New("k8s"),
		WorkspaceID:                   req.WorkspaceID,
		ProjectID:                     req.ProjectID,
		EnvironmentID:                 req.EnvironmentID,
		GitCommitSha:                  sql.NullString{String: "", Valid: false},
		GitBranch:                     sql.NullString{String: "", Valid: false},
		SentinelConfig:                []byte("{}"),
		GitCommitMessage:              sql.NullString{String: "", Valid: false},
		GitCommitAuthorHandle:         sql.NullString{String: "", Valid: false},
		GitCommitAuthorAvatarUrl:      sql.NullString{String: "", Valid: false},
		GitCommitTimestamp:            sql.NullInt64{Int64: 0, Valid: false},
		OpenapiSpec:                   sql.NullString{String: "", Valid: false},
		EncryptedEnvironmentVariables: []byte("{}"),
		Command:                       nil,
		Status:                        req.Status,
		CpuMillicores:                 256,
		MemoryMib:                     256,
		CreatedAt:                     createdAt,
		UpdatedAt:                     req.UpdatedAt,
		Port:                          8080,
		ShutdownSignal:                db.DeploymentsShutdownSignalSIGINT,
		Healthcheck:                   dbtype.NullHealthcheck{Healthcheck: nil, Valid: false},
	})
	require.NoError(s.t, err)

	deployment, err := db.Query.FindDeploymentById(ctx, s.DB.RO(), id)
	require.NoError(s.t, err)

	return deployment
}

// CreateRootKey creates a root key with optional permissions
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
	ForWorkspaceID *string // For creating root keys that target a specific workspace

	Recoverable bool

	RefillAmount *int32
	RefillDay    *int16

	Permissions []CreatePermissionRequest
	Roles       []CreateRoleRequest
	Ratelimits  []CreateRatelimitRequest
}

type CreateKeyResponse struct {
	KeyID string
	Key   string

	RolesIds      []string
	PermissionIds []string
}

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
		encryption, encryptErr := s.Vault.Encrypt(ctx, connect.NewRequest(&vaultv1.EncryptRequest{
			Keyring: req.WorkspaceID,
			Data:    key,
		}))
		require.NoError(s.t, encryptErr)
		err = db.Query.InsertKeyEncryption(ctx, s.DB.RW(), db.InsertKeyEncryptionParams{
			WorkspaceID:     req.WorkspaceID,
			KeyID:           keyID,
			CreatedAt:       time.Now().UnixMilli(),
			Encrypted:       encryption.Msg.GetEncrypted(),
			EncryptionKeyID: encryption.Msg.GetKeyId(),
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

type CreateRatelimitRequest struct {
	Name        string
	WorkspaceID string
	AutoApply   bool
	Duration    int64
	Limit       int32
	IdentityID  *string
	KeyID       *string
}

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

type CreateIdentityRequest struct {
	WorkspaceID string
	ExternalID  string
	Meta        []byte
	Ratelimits  []CreateRatelimitRequest
}

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

type CreateRoleRequest struct {
	Name        string
	Description *string
	WorkspaceID string

	Permissions []CreatePermissionRequest
}

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

type CreatePermissionRequest struct {
	Name        string
	Slug        string
	Description *string
	WorkspaceID string
}

// CreateWorkspaceWithQuotaRequest configures the workspace and quota to create.
type CreateWorkspaceWithQuotaRequest struct {
	// RequestsPerMonth is the maximum number of requests allowed per month.
	// Use 0 or negative to skip quota creation.
	RequestsPerMonth int64
	// LogsRetentionDays is the number of days to retain logs. Defaults to 0.
	LogsRetentionDays int32
	// AuditLogsRetentionDays is the number of days to retain audit logs. Defaults to 0.
	AuditLogsRetentionDays int32
	// Team indicates if the workspace has team features enabled.
	Team bool
}

// CreateWorkspaceWithQuota creates a workspace with an associated quota.
// Returns the created db.Workspace for use in tests.
func (s *Seeder) CreateWorkspaceWithQuota(ctx context.Context, req CreateWorkspaceWithQuotaRequest) db.Workspace {
	ws := s.CreateWorkspace(ctx)

	if req.RequestsPerMonth > 0 {
		err := db.Query.UpsertQuota(ctx, s.DB.RW(), db.UpsertQuotaParams{
			WorkspaceID:            ws.ID,
			RequestsPerMonth:       req.RequestsPerMonth,
			AuditLogsRetentionDays: req.AuditLogsRetentionDays,
			LogsRetentionDays:      req.LogsRetentionDays,
			Team:                   req.Team,
		})
		require.NoError(s.t, err)
	}

	return ws
}

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
