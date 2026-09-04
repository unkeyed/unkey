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
	"github.com/unkeyed/unkey/gen/rpc/vault"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/hash"
	dbtype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
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
	Vault     vault.VaultServiceClient
	Resources Resources

	// workspaceIDs are the workspaces this seeder created, deleted with
	// everything hanging off them when the test ends.
	workspaceIDs []string
}

// New creates a new Seeder instance
func New(t *testing.T, database db.Database, vault vault.VaultServiceClient) *Seeder {
	s := &Seeder{
		t:            t,
		DB:           database,
		Vault:        vault,
		Resources:    Resources{}, //nolint:exhaustruct
		workspaceIDs: nil,
	}
	t.Cleanup(s.cleanup)
	return s
}

// cleanup removes every workspace this seeder created, and the projects, apps,
// environments and deployments belonging to them.
//
// Integration tests share one MySQL container across test processes and across
// runs, and the ctrl crons scan the whole database rather than one workspace:
// the idle-preview scan pages over every preview environment, and
// the usage and billing handlers walk every workspace. Rows a test leaves
// behind are therefore rescanned by every later run, which makes each run
// slower until a scan outlives the harness timeout. Deleting only the ids this
// seeder created keeps that safe while other test binaries use the same
// database.
func (s *Seeder) cleanup() {
	if len(s.workspaceIDs) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	require.NoError(s.t, s.DB.DeleteWorkspacesWithChildren(ctx, s.workspaceIDs))
}

func (s *Seeder) CreateWorkspace(ctx context.Context) db.Workspace {
	params := db.InsertWorkspaceParams{
		ID:           uid.New("test_ws"),
		OrgID:        uid.New("test_org"),
		Name:         uid.New("test_name"),
		Slug:         uid.New("slug"),
		CreatedAt:    time.Now().UnixMilli(),
		K8sNamespace: sql.NullString{Valid: true, String: uid.DNS1035()},
	}

	err := s.DB.InsertWorkspace(ctx, params)
	require.NoError(s.t, err)
	s.workspaceIDs = append(s.workspaceIDs, params.ID)

	s.CreateProject(ctx, CreateProjectRequest{
		ID:               uid.New(uid.ProjectPrefix),
		WorkspaceID:      params.ID,
		Name:             "Default",
		Slug:             "default",
		DeleteProtection: true,
	})

	err = s.DB.InsertWorkspaceBilling(ctx, db.InsertWorkspaceBillingParams{
		WorkspaceID: params.ID,
		CreatedAt:   time.Now().UnixMilli(),
	})
	require.NoError(s.t, err)

	err = s.DB.UpsertLimit(ctx, db.UpsertLimitParams{
		WorkspaceID:                           params.ID,
		ApiBillableOperationsCountMaxPerMonth: 1_000_000,
		ApiRequestsCountMaxPerMinute:          sql.NullInt32{}, //nolint:exhaustruct
		LogsRetentionDaysMax:                  30,
		LogsAuditRetentionDaysMax:             30,
		TeamEnabled:                           false,
		CpuCoresMax:                           10,
		CpuCoresMaxPerInstance:                2,
		MemoryMibMax:                          20_480,
		MemoryMibMaxPerInstance:               4_096,
		StorageMibMax:                         51_200,
		StorageMibMaxPerInstance:              10_240,
		BuildsConcurrentMax:                   1,
		CustomDomainsMax:                      0,
		AutoscalingReplicasMax:                0,
	})
	require.NoError(s.t, err)

	ws, err := s.DB.FindWorkspaceByID(ctx, params.ID)
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
	keySpace, err := s.DB.FindKeySpaceByID(ctx, s.Resources.RootApi.KeyAuthID.String)
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
	projectID := s.defaultProjectID(ctx, req.WorkspaceID)
	keySpaceID := uid.New(uid.KeySpacePrefix)
	err := s.DB.InsertKeySpace(ctx, db.InsertKeySpaceParams{
		ID:                 keySpaceID,
		WorkspaceID:        req.WorkspaceID,
		ProjectID:          projectID,
		CreatedAtM:         time.Now().UnixMilli(),
		DefaultPrefix:      sql.NullString{String: ptr.SafeDeref(req.DefaultPrefix), Valid: req.DefaultPrefix != nil},
		DefaultBytes:       sql.NullInt32{Int32: ptr.SafeDeref(req.DefaultBytes), Valid: req.DefaultBytes != nil},
		StoreEncryptedKeys: req.EncryptedKeys,
	})
	require.NoError(s.t, err)

	apiID := uid.New("api")
	err = s.DB.InsertApi(ctx, db.InsertApiParams{
		ID:          apiID,
		Name:        ptr.SafeDeref(req.Name, "test-api"),
		WorkspaceID: req.WorkspaceID,
		ProjectID:   projectID,
		IpWhitelist: sql.NullString{String: req.IpWhitelist, Valid: req.IpWhitelist != ""},
		AuthType:    db.NullApisAuthType{Valid: true, ApisAuthType: db.ApisAuthTypeKey},
		KeyAuthID:   sql.NullString{Valid: true, String: keySpaceID},
		CreatedAtM:  ptr.SafeDeref(req.CreatedAt, time.Now().UnixMilli()),
	})
	require.NoError(s.t, err)

	api, err := s.DB.FindApiByID(ctx, apiID)
	require.NoError(s.t, err)

	return api
}

// defaultProjectID returns the exact default project established by CreateWorkspace.
func (s *Seeder) defaultProjectID(ctx context.Context, workspaceID string) string {
	projectID, err := s.DB.FindDefaultProjectByWorkspaceID(ctx, workspaceID)
	require.NoError(s.t, err)
	require.NotEmpty(s.t, projectID)
	return projectID
}

type CreateProjectRequest struct {
	ID               string
	WorkspaceID      string
	Name             string
	Slug             string
	DeleteProtection bool
}

func (h *Seeder) CreateProject(ctx context.Context, req CreateProjectRequest) db.Project {
	err := h.DB.InsertProject(ctx, db.InsertProjectParams{
		ID:               req.ID,
		WorkspaceID:      req.WorkspaceID,
		Name:             req.Name,
		Slug:             req.Slug,
		DeleteProtection: sql.NullBool{Valid: true, Bool: req.DeleteProtection},
		CreatedAt:        time.Now().UnixMilli(),
		UpdatedAt:        sql.NullInt64{Int64: 0, Valid: false},
	})
	require.NoError(h.t, err)

	project, err := h.DB.FindProjectById(ctx, req.ID)
	require.NoError(h.t, err)

	return db.Project{
		ID:               project.ID,
		WorkspaceID:      project.WorkspaceID,
		Name:             project.Name,
		Slug:             project.Slug,
		DeleteProtection: project.DeleteProtection,
		CreatedAt:        project.CreatedAt,
		UpdatedAt:        project.UpdatedAt,
		Pk:               0,
		DepotProjectID:   sql.NullString{String: "", Valid: false},
	}
}

type CreateEnvironmentRequest struct {
	ID               string
	WorkspaceID      string
	ProjectID        string
	AppID            string
	Slug             string
	Description      string
	Kind             dbtype.EnvironmentKind
	SentinelConfig   []byte
	DeleteProtection bool
}

func (s *Seeder) CreateEnvironment(ctx context.Context, req CreateEnvironmentRequest) db.Environment {
	now := time.Now().UnixMilli()
	kind := req.Kind
	if kind == "" {
		kind = dbtype.EnvironmentKindPreview
	}

	err := s.DB.InsertEnvironment(ctx, db.InsertEnvironmentParams{
		ID:          req.ID,
		WorkspaceID: req.WorkspaceID,
		ProjectID:   req.ProjectID,
		AppID:       req.AppID,
		Slug:        req.Slug,
		Description: req.Description,
		Kind:        kind,
		CreatedAt:   now,
		UpdatedAt:   sql.NullInt64{Int64: 0, Valid: false},
	})
	require.NoError(s.t, err)

	// Insert default app build settings for this (app, environment) pair.
	err = s.DB.UpsertAppBuildSettings(ctx, db.UpsertAppBuildSettingsParams{
		WorkspaceID:   req.WorkspaceID,
		AppID:         req.AppID,
		EnvironmentID: req.ID,
		Dockerfile:    sql.NullString{Valid: true, String: "Dockerfile"},
		DockerContext: ".",
		BuildCommand:  sql.NullString{Valid: false, String: ""},
		WatchPaths:    nil,
		AutoDeploy:    true,
		CreatedAt:     now,
		UpdatedAt:     sql.NullInt64{Valid: false},
	})
	require.NoError(s.t, err)

	// Insert default app runtime settings for this (app, environment) pair.
	err = s.DB.UpsertAppRuntimeSettings(ctx, db.UpsertAppRuntimeSettingsParams{
		WorkspaceID:      req.WorkspaceID,
		AppID:            req.AppID,
		EnvironmentID:    req.ID,
		Port:             8080,
		CpuMillicores:    250,
		MemoryMib:        256,
		StorageMib:       0,
		Command:          nil,
		Healthcheck:      dbtype.NullHealthcheck{Healthcheck: nil, Valid: false},
		ShutdownSignal:   db.AppRuntimeSettingsShutdownSignalSIGTERM,
		UpstreamProtocol: db.AppRuntimeSettingsUpstreamProtocolHttp1,
		SentinelConfig:   []byte("{}"),
		OpenapiSpecPath:  sql.NullString{Valid: false},
		CreatedAt:        now,
		UpdatedAt:        sql.NullInt64{Valid: false},
	})
	require.NoError(s.t, err)

	environment, err := s.DB.FindEnvironmentById(ctx, req.ID)
	require.NoError(s.t, err)

	return db.Environment{
		Pk:               0,
		ID:               environment.ID,
		WorkspaceID:      environment.WorkspaceID,
		ProjectID:        environment.ProjectID,
		AppID:            req.AppID,
		Slug:             environment.Slug,
		Description:      req.Description,
		Kind:             environment.Kind,
		DeleteProtection: sql.NullBool{Valid: true, Bool: req.DeleteProtection},
		CreatedAt:        now,
		UpdatedAt:        sql.NullInt64{Int64: 0, Valid: false},
	}
}

type CreateAppRequest struct {
	ID            string
	WorkspaceID   string
	ProjectID     string
	Name          string
	Slug          string
	DefaultBranch string
}

func (s *Seeder) CreateApp(ctx context.Context, req CreateAppRequest) db.App {
	now := time.Now().UnixMilli()

	err := s.DB.InsertApp(ctx, db.InsertAppParams{
		ID:               req.ID,
		WorkspaceID:      req.WorkspaceID,
		ProjectID:        req.ProjectID,
		Name:             req.Name,
		Slug:             req.Slug,
		DefaultBranch:    req.DefaultBranch,
		DeleteProtection: sql.NullBool{Valid: true, Bool: false},
		CreatedAt:        now,
		UpdatedAt:        sql.NullInt64{Valid: false},
	})
	require.NoError(s.t, err)

	app, err := s.DB.FindAppById(ctx, req.ID)
	require.NoError(s.t, err)

	return app
}

// CreateAppWithSettings creates an app plus its build and runtime settings for a given environment.
func (s *Seeder) CreateAppWithSettings(ctx context.Context, req CreateAppRequest, environmentID string) db.App {
	app := s.CreateApp(ctx, req)
	now := time.Now().UnixMilli()

	// Seed default build settings
	err := s.DB.UpsertAppBuildSettings(ctx, db.UpsertAppBuildSettingsParams{
		WorkspaceID:   req.WorkspaceID,
		AppID:         req.ID,
		EnvironmentID: environmentID,
		Dockerfile:    sql.NullString{Valid: false, String: ""},
		DockerContext: "",
		BuildCommand:  sql.NullString{Valid: false, String: ""},
		WatchPaths:    nil,
		AutoDeploy:    true,
		CreatedAt:     now,
		UpdatedAt:     sql.NullInt64{Valid: false},
	})
	require.NoError(s.t, err)

	// Seed default runtime settings
	err = s.DB.UpsertAppRuntimeSettings(ctx, db.UpsertAppRuntimeSettingsParams{
		WorkspaceID:      req.WorkspaceID,
		AppID:            req.ID,
		EnvironmentID:    environmentID,
		Port:             8080,
		CpuMillicores:    250,
		MemoryMib:        256,
		StorageMib:       0,
		Command:          nil,
		Healthcheck:      dbtype.NullHealthcheck{Healthcheck: nil, Valid: false},
		ShutdownSignal:   db.AppRuntimeSettingsShutdownSignalSIGTERM,
		UpstreamProtocol: db.AppRuntimeSettingsUpstreamProtocolHttp1,
		SentinelConfig:   []byte("{}"),
		OpenapiSpecPath:  sql.NullString{Valid: false},
		CreatedAt:        now,
		UpdatedAt:        sql.NullInt64{Valid: false},
	})
	require.NoError(s.t, err)

	return app
}

type CreateDeploymentRequest struct {
	ID            string
	WorkspaceID   string
	ProjectID     string
	AppID         string
	EnvironmentID string
	Status        dbtype.DeploymentsStatus
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

	err := s.DB.InsertDeployment(ctx, db.InsertDeploymentParams{
		ID:                            id,
		K8sName:                       uid.New("k8s"),
		WorkspaceID:                   req.WorkspaceID,
		ProjectID:                     req.ProjectID,
		AppID:                         req.AppID,
		EnvironmentID:                 req.EnvironmentID,
		Source:                        db.DeploymentsSourceUnknown,
		ImageRequested:                sql.NullString{Valid: false},
		GitCommitSha:                  sql.NullString{String: "", Valid: false},
		GitBranch:                     sql.NullString{String: "", Valid: false},
		SentinelConfig:                []byte("{}"),
		GitCommitMessage:              sql.NullString{String: "", Valid: false},
		GitCommitAuthorHandle:         sql.NullString{String: "", Valid: false},
		GitCommitAuthorAvatarUrl:      sql.NullString{String: "", Valid: false},
		GitCommitTimestamp:            sql.NullInt64{Int64: 0, Valid: false},
		EncryptedEnvironmentVariables: []byte("{}"),
		Command:                       nil,
		Status:                        req.Status,
		CpuMillicores:                 250,
		MemoryMib:                     256,
		StorageMib:                    0,
		CreatedAt:                     createdAt,
		UpdatedAt:                     req.UpdatedAt,
		Port:                          8080,
		ShutdownSignal:                db.DeploymentsShutdownSignalSIGINT,
		UpstreamProtocol:              db.DeploymentsUpstreamProtocolHttp1,
		Healthcheck:                   dbtype.NullHealthcheck{Healthcheck: nil, Valid: false},
		PrNumber:                      sql.NullInt64{Int64: 0, Valid: false},
		ForkRepositoryFullName:        sql.NullString{String: "", Valid: false},
		DeploymentTrigger:             db.DeploymentsTriggerUnknown,
		TriggeredBy:                   sql.NullString{Valid: false},
		TriggerReason:                 sql.NullString{Valid: false},
	})
	require.NoError(s.t, err)

	deployment, err := s.DB.FindDeploymentById(ctx, id)
	require.NoError(s.t, err)

	return deployment
}

// CreateRootKey creates a root key with optional permissions
func (s *Seeder) CreateRootKey(ctx context.Context, workspaceID string, permissions ...string) string {
	key := uid.New("test_root_key")

	insertKeyParams := db.InsertKeyParams{
		ID:                 uid.New("test_root_key"),
		Hash:               hash.Sha256(key),
		Prefix:             "",
		WorkspaceID:        s.Resources.RootWorkspace.ID,
		ForWorkspaceID:     sql.NullString{String: workspaceID, Valid: true},
		KeySpaceID:         s.Resources.RootKeySpace.ID,
		Start:              key[:4],
		End:                key[len(key)-4:],
		CreatedAtM:         time.Now().UnixMilli(),
		Enabled:            true,
		Name:               sql.NullString{String: "", Valid: false},
		IdentityID:         sql.NullString{String: "", Valid: false},
		Meta:               sql.NullString{String: "", Valid: false},
		Expires:            sql.NullTime{Time: time.Time{}, Valid: false},
		RemainingRequests:  sql.NullInt64{Int64: 0, Valid: false},
		RefillDay:          sql.NullInt16{Int16: 0, Valid: false},
		RefillAmount:       sql.NullInt64{Int64: 0, Valid: false},
		PendingMigrationID: sql.NullString{Valid: false, String: ""},
	}

	err := s.DB.InsertKey(ctx, insertKeyParams)
	require.NoError(s.t, err)

	if len(permissions) > 0 {
		projectID := s.defaultProjectID(ctx, s.Resources.RootWorkspace.ID)
		for _, permission := range permissions {
			permissionID := uid.New(uid.TestPrefix)
			err := s.DB.InsertPermission(ctx, db.InsertPermissionParams{
				PermissionID: permissionID,
				WorkspaceID:  s.Resources.RootWorkspace.ID,
				ProjectID:    projectID,
				Name:         permission,
				Slug:         permission,
				Description:  dbtype.NullString{String: "", Valid: false},
				CreatedAtM:   time.Now().UnixMilli(),
			})

			mysqlErr := &mysql.MySQLError{} // nolint:exhaustruct
			if errors.As(err, &mysqlErr) {
				require.True(s.t, db.IsDuplicateKeyError(err), "Expected duplicate key error, got MySQL error number %d", mysqlErr.Number)
				existing, findErr := s.DB.FindPermissionBySlugAndProjectID(ctx, db.FindPermissionBySlugAndProjectIDParams{
					WorkspaceID: s.Resources.RootWorkspace.ID,
					ProjectID:   projectID,
					Slug:        permission,
				})
				require.NoError(s.t, findErr)
				permissionID = existing.ID

			} else {
				require.NoError(s.t, err)
			}

			err = s.DB.InsertKeyPermission(ctx, db.InsertKeyPermissionParams{
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
	Remaining      *int64
	IdentityID     *string
	Meta           *string
	Expires        *time.Time
	Name           *string
	Deleted        bool
	ForWorkspaceID *string // For creating root keys that target a specific workspace

	Recoverable bool

	RefillAmount *int64
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

	err := s.DB.InsertKey(ctx, db.InsertKeyParams{
		ID:                 keyID,
		KeySpaceID:         req.KeySpaceID,
		WorkspaceID:        req.WorkspaceID,
		CreatedAtM:         time.Now().UnixMilli(),
		Hash:               hash.Sha256(key),
		Prefix:             "",
		Enabled:            !req.Disabled,
		Start:              start,
		End:                key[len(key)-4:],
		Name:               sql.NullString{String: ptr.SafeDeref(req.Name, "test-key"), Valid: true},
		ForWorkspaceID:     sql.NullString{String: ptr.SafeDeref(req.ForWorkspaceID, ""), Valid: req.ForWorkspaceID != nil},
		Meta:               sql.NullString{String: ptr.SafeDeref(req.Meta, ""), Valid: req.Meta != nil},
		IdentityID:         sql.NullString{String: ptr.SafeDeref(req.IdentityID, ""), Valid: req.IdentityID != nil},
		Expires:            sql.NullTime{Time: ptr.SafeDeref(req.Expires, time.Time{}), Valid: req.Expires != nil},
		RemainingRequests:  sql.NullInt64{Int64: ptr.SafeDeref(req.Remaining, 0), Valid: req.Remaining != nil},
		RefillAmount:       sql.NullInt64{Int64: ptr.SafeDeref(req.RefillAmount, 0), Valid: req.RefillAmount != nil},
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
		err = s.DB.SoftDeleteKeyByID(ctx, db.SoftDeleteKeyByIDParams{
			Now: sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
			ID:  keyID,
		})

		require.NoError(s.t, err)
	}

	if req.Recoverable && s.Vault != nil {
		encryption, encryptErr := s.Vault.Encrypt(ctx, &vaultv1.EncryptRequest{
			Keyring: req.WorkspaceID,
			Data:    key,
		})
		require.NoError(s.t, encryptErr)
		err = s.DB.InsertKeyEncryption(ctx, db.InsertKeyEncryptionParams{
			WorkspaceID:     req.WorkspaceID,
			KeyID:           keyID,
			CreatedAt:       time.Now().UnixMilli(),
			Encrypted:       encryption.GetEncrypted(),
			EncryptionKeyID: encryption.GetKeyId(),
		})
		require.NoError(s.t, err)
	}

	for _, role := range req.Roles {
		roleID := s.CreateRole(ctx, role)
		err = s.DB.InsertKeyRole(ctx, db.InsertKeyRoleParams{
			KeyID:       keyID,
			RoleID:      roleID,
			WorkspaceID: req.WorkspaceID,
			CreatedAtM:  time.Now().UnixMilli(),
		})
		require.NoError(s.t, err)
		res.RolesIds = append(res.RolesIds, roleID)
	}

	for _, permission := range req.Permissions {
		perm := s.CreatePermission(ctx, permission)
		err = s.DB.InsertKeyPermission(ctx, db.InsertKeyPermissionParams{
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
	Duration    uint64
	Limit       uint64
	IdentityID  *string
	KeyID       *string
}

func (s *Seeder) CreateRatelimit(ctx context.Context, req CreateRatelimitRequest) string {
	ratelimitID := uid.New(uid.RatelimitPrefix)
	createdAt := time.Now().UnixMilli()
	var err error

	if req.IdentityID != nil {
		err = s.DB.InsertIdentityRatelimit(ctx, db.InsertIdentityRatelimitParams{
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
		err = s.DB.InsertKeyRatelimit(ctx, db.InsertKeyRatelimitParams{
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

	return ratelimitID
}

type CreateIdentityRequest struct {
	WorkspaceID string
	ExternalID  string
	Meta        []byte
	Ratelimits  []CreateRatelimitRequest
}

func (s *Seeder) CreateIdentity(ctx context.Context, req CreateIdentityRequest) string {
	projectID := s.defaultProjectID(ctx, req.WorkspaceID)
	metaBytes := []byte("{}")
	if len(req.Meta) > 0 {
		metaBytes = req.Meta
	}

	require.NoError(s.t, assert.NotEmpty(req.ExternalID, "Identity ExternalID must be set"))
	require.NoError(s.t, assert.NotEmpty(req.WorkspaceID, "Identity WorkspaceID must be set"))

	identityID := uid.New(uid.IdentityPrefix)
	err := s.DB.InsertIdentity(ctx, db.InsertIdentityParams{
		ID:          identityID,
		ExternalID:  req.ExternalID,
		WorkspaceID: req.WorkspaceID,
		ProjectID:   projectID,
		Environment: "",
		CreatedAt:   time.Now().UnixMilli(),
		Meta:        metaBytes,
	})
	require.NoError(s.t, err)

	for _, ratelimit := range req.Ratelimits {
		ratelimit.IdentityID = ptr.P(identityID)
		s.CreateRatelimit(ctx, ratelimit)
	}

	return identityID
}

type CreateRoleRequest struct {
	Name        string
	Description *string
	WorkspaceID string

	Permissions []CreatePermissionRequest
}

func (s *Seeder) CreateRole(ctx context.Context, req CreateRoleRequest) string {
	projectID := s.defaultProjectID(ctx, req.WorkspaceID)
	require.NoError(s.t, assert.NotEmpty(req.WorkspaceID, "Role WorkspaceID must be set"))
	require.NoError(s.t, assert.NotEmpty(req.Name, "Role Name must be set"))

	roleID := uid.New(uid.RolePrefix)
	createdAt := time.Now().UnixMilli()

	err := s.DB.InsertRole(ctx, db.InsertRoleParams{
		RoleID:      roleID,
		WorkspaceID: req.WorkspaceID,
		ProjectID:   projectID,
		Name:        req.Name,
		CreatedAt:   createdAt,
		Description: sql.NullString{Valid: req.Description != nil, String: ptr.SafeDeref(req.Description, "")},
	})
	require.NoError(s.t, err)

	for _, permission := range req.Permissions {
		perm := s.CreatePermission(ctx, permission)
		err = s.DB.InsertRolePermission(ctx, db.InsertRolePermissionParams{
			RoleID:       roleID,
			PermissionID: perm.ID,
			WorkspaceID:  req.WorkspaceID,
			CreatedAtM:   time.Now().UnixMilli(),
		})
		require.NoError(s.t, err)
	}

	return roleID
}

type CreatePermissionRequest struct {
	Name        string
	Slug        string
	Description *string
	WorkspaceID string
}

// CreateWorkspaceWithLimitsRequest configures the workspace and limits to create.
type CreateWorkspaceWithLimitsRequest struct {
	// RequestsPerMonth is the maximum number of requests allowed per month.
	// Use 0 or negative to skip limit creation.
	RequestsPerMonth int64
	// LogsRetentionDays is the number of days to retain logs. Defaults to 0.
	LogsRetentionDays int32
	// AuditLogsRetentionDays is the number of days to retain audit logs. Defaults to 0.
	AuditLogsRetentionDays int32
	// Team indicates if the workspace has team features enabled.
	Team bool
}

// CreateWorkspaceWithLimits creates a workspace with associated limits.
// Returns the created db.Workspace for use in tests.
func (s *Seeder) CreateWorkspaceWithLimits(ctx context.Context, req CreateWorkspaceWithLimitsRequest) db.Workspace {
	ws := s.CreateWorkspace(ctx)

	if req.RequestsPerMonth > 0 {
		err := s.DB.UpsertLimit(ctx, db.UpsertLimitParams{
			WorkspaceID:                           ws.ID,
			ApiBillableOperationsCountMaxPerMonth: uint64(req.RequestsPerMonth),
			ApiRequestsCountMaxPerMinute:          sql.NullInt32{}, //nolint:exhaustruct
			LogsRetentionDaysMax:                  uint16(req.LogsRetentionDays),
			LogsAuditRetentionDaysMax:             uint16(req.AuditLogsRetentionDays),
			TeamEnabled:                           req.Team,
			CpuCoresMax:                           10,
			CpuCoresMaxPerInstance:                2,
			MemoryMibMax:                          20_480,
			MemoryMibMaxPerInstance:               4_096,
			StorageMibMax:                         51_200,
			StorageMibMaxPerInstance:              10_240,
			BuildsConcurrentMax:                   1,
			CustomDomainsMax:                      0,
			AutoscalingReplicasMax:                0,
		})
		require.NoError(s.t, err)
	}

	return ws
}

func (s *Seeder) CreatePermission(ctx context.Context, req CreatePermissionRequest) db.Permission {
	projectID := s.defaultProjectID(ctx, req.WorkspaceID)
	require.NoError(s.t, assert.NotEmpty(req.WorkspaceID, "Permission WorkspaceID must be set"))
	require.NoError(s.t, assert.NotEmpty(req.Name, "Permission Name must be set"))
	require.NoError(s.t, assert.NotEmpty(req.Slug, "Permission Slug must be set"))

	permissionID := uid.New(uid.PermissionPrefix)
	createdAt := time.Now().UnixMilli()

	err := s.DB.InsertPermission(ctx, db.InsertPermissionParams{
		PermissionID: permissionID,
		WorkspaceID:  req.WorkspaceID,
		ProjectID:    projectID,
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
		ProjectID:   projectID,
		Name:        req.Name,
		Slug:        req.Slug,
		Description: dbtype.NullString{Valid: req.Description != nil, String: ptr.SafeDeref(req.Description, "")},
		CreatedAtM:  createdAt,
		UpdatedAtM:  sql.NullInt64{Valid: false, Int64: 0},
	}
}

type CreateRegionRequest struct {
	Name     string
	Platform string
}

func (s *Seeder) CreateRegion(ctx context.Context, req CreateRegionRequest) db.Region {
	id := uid.New(uid.RegionPrefix)

	err := s.DB.UpsertRegion(ctx, db.UpsertRegionParams{
		ID:       id,
		Name:     req.Name,
		Platform: req.Platform,
	})
	require.NoError(s.t, err)

	region, err := s.DB.FindRegionByPlatformAndName(ctx, db.FindRegionByPlatformAndNameParams{
		Platform: req.Platform,
		Name:     req.Name,
	})
	require.NoError(s.t, err)

	return region
}

type CreateInstanceRequest struct {
	DeploymentID string
	WorkspaceID  string
	ProjectID    string
	AppID        string
	RegionID     string
	Address      string
}

func (s *Seeder) CreateInstance(ctx context.Context, req CreateInstanceRequest) db.Instance {
	id := uid.New("inst")
	k8sName := uid.New("k8s")

	err := s.DB.UpsertInstance(ctx, db.UpsertInstanceParams{
		ID:            id,
		DeploymentID:  req.DeploymentID,
		WorkspaceID:   req.WorkspaceID,
		ProjectID:     req.ProjectID,
		AppID:         req.AppID,
		RegionID:      req.RegionID,
		K8sName:       k8sName,
		Address:       req.Address,
		CpuMillicores: 100,
		MemoryMib:     128,
		Status:        db.InstancesStatusRunning,
	})
	require.NoError(s.t, err)

	return db.Instance{
		Pk:            0,
		ID:            id,
		DeploymentID:  req.DeploymentID,
		WorkspaceID:   req.WorkspaceID,
		ProjectID:     req.ProjectID,
		AppID:         req.AppID,
		RegionID:      req.RegionID,
		K8sName:       k8sName,
		Address:       req.Address,
		CpuMillicores: 100,
		MemoryMib:     128,
		StorageMib:    0,
		Status:        db.InstancesStatusRunning,
		// Mirrors the column default applied by UpsertInstance: a fresh
		// instance has restartCount=0 and no terminations or waiting
		// reasons. ctrl's RecordInstanceExit / RecordInstanceCrashLoopBackOff
		// keep this in sync as events arrive.
		ContainerStatus: dbtype.ContainerStatus{
			RestartCount:         0,
			LastTerminationState: nil,
			Waiting:              nil,
		},
	}
}
