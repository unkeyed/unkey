package seed

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
	vaultv1 "github.com/unkeyed/unkey/go/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/go/pkg/assert"
	"github.com/unkeyed/unkey/go/pkg/db"
	dbtype "github.com/unkeyed/unkey/go/pkg/db/types"
	"github.com/unkeyed/unkey/go/pkg/hash"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/uid"
	"github.com/unkeyed/unkey/go/pkg/vault"
)

// Resources represents seed data created for tests
type Resources struct {
	RootWorkspace db.Workspace
	RootKeyring   db.KeyAuth
	RootApi       db.Api
	UserWorkspace db.Workspace
}

// Seeder provides methods to seed test data
type Seeder struct {
	t         *testing.T
	DB        db.Database
	Vault     *vault.Service
	Resources Resources
}

// New creates a new Seeder instance
func New(t *testing.T, database db.Database, vault *vault.Service) *Seeder {
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
	keyring, err := db.Query.FindKeyringByID(ctx, s.DB.RW(), s.Resources.RootApi.KeyAuthID.String)
	require.NoError(s.t, err)
	s.Resources.RootKeyring = keyring
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
	keyAuthID := uid.New(uid.KeyAuthPrefix)
	err := db.Query.InsertKeyring(ctx, s.DB.RW(), db.InsertKeyringParams{
		ID:                 keyAuthID,
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
		KeyAuthID:   sql.NullString{Valid: true, String: keyAuthID},
		CreatedAtM:  ptr.SafeDeref(req.CreatedAt, time.Now().UnixMilli()),
	})
	require.NoError(s.t, err)

	api, err := db.Query.FindApiByID(ctx, s.DB.RW(), apiID)
	require.NoError(s.t, err)

	return api
}

// CreateRootKey creates a root key with optional permissions
func (s *Seeder) CreateRootKey(ctx context.Context, workspaceID string, permissions ...string) string {
	key := uid.New("test_root_key")

	insertKeyParams := db.InsertKeyParams{
		ID:                uid.New("test_root_key"),
		Hash:              hash.Sha256(key),
		WorkspaceID:       s.Resources.RootWorkspace.ID,
		ForWorkspaceID:    sql.NullString{String: workspaceID, Valid: true},
		KeyringID:         s.Resources.RootKeyring.ID,
		Start:             key[:4],
		CreatedAtM:        time.Now().UnixMilli(),
		Enabled:           true,
		Name:              sql.NullString{String: "", Valid: false},
		IdentityID:        sql.NullString{String: "", Valid: false},
		Meta:              sql.NullString{String: "", Valid: false},
		Expires:           sql.NullTime{Time: time.Time{}, Valid: false},
		RemainingRequests: sql.NullInt32{Int32: 0, Valid: false},
		RefillDay:         sql.NullInt16{Int16: 0, Valid: false},
		RefillAmount:      sql.NullInt32{Int32: 0, Valid: false},
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
			})
			require.NoError(s.t, err)
		}
	}

	return key
}

type CreateKeyRequest struct {
	Disabled    bool
	WorkspaceID string
	KeyAuthID   string
	Remaining   *int32
	IdentityID  *string
	Meta        *string
	Expires     *time.Time
	Name        *string
	Deleted     bool

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
		ID:                keyID,
		KeyringID:         req.KeyAuthID,
		WorkspaceID:       req.WorkspaceID,
		CreatedAtM:        time.Now().UnixMilli(),
		Hash:              hash.Sha256(key),
		Enabled:           !req.Disabled,
		Start:             start,
		Name:              sql.NullString{String: ptr.SafeDeref(req.Name, "test-key"), Valid: true},
		ForWorkspaceID:    sql.NullString{String: "", Valid: false},
		Meta:              sql.NullString{String: ptr.SafeDeref(req.Meta, ""), Valid: req.Meta != nil},
		IdentityID:        sql.NullString{String: ptr.SafeDeref(req.IdentityID, ""), Valid: req.IdentityID != nil},
		Expires:           sql.NullTime{Time: ptr.SafeDeref(req.Expires, time.Time{}), Valid: req.Expires != nil},
		RemainingRequests: sql.NullInt32{Int32: ptr.SafeDeref(req.Remaining, 0), Valid: req.Remaining != nil},
		RefillAmount:      sql.NullInt32{Int32: ptr.SafeDeref(req.RefillAmount, 0), Valid: req.RefillAmount != nil},
		RefillDay:         sql.NullInt16{Int16: ptr.SafeDeref(req.RefillDay, 0), Valid: req.RefillDay != nil},
	})
	require.NoError(s.t, err)

	res := CreateKeyResponse{
		KeyID: keyID,
		Key:   key,
	}

	if req.Deleted {
		err = db.Query.SoftDeleteKeyByID(ctx, s.DB.RW(), db.SoftDeleteKeyByIDParams{
			Now: sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
			ID:  keyID,
		})

		require.NoError(s.t, err)
	}

	if req.Recoverable && s.Vault != nil {
		encryption, err := s.Vault.Encrypt(ctx, &vaultv1.EncryptRequest{
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
		roleID := s.CreateRole(ctx, role)
		err = db.Query.InsertKeyRole(ctx, s.DB.RW(), db.InsertKeyRoleParams{
			KeyID:       keyID,
			RoleID:      roleID,
			WorkspaceID: req.WorkspaceID,
			CreatedAtM:  time.Now().UnixMilli(),
		})
		require.NoError(s.t, err)
		res.RolesIds = append(res.RolesIds, roleID)
	}

	for _, permission := range req.Permissions {
		permissionID := s.CreatePermission(ctx, permission)
		err = db.Query.InsertKeyPermission(ctx, s.DB.RW(), db.InsertKeyPermissionParams{
			KeyID:        keyID,
			PermissionID: permissionID,
			WorkspaceID:  req.WorkspaceID,
			CreatedAt:    time.Now().UnixMilli(),
		})

		require.NoError(s.t, err)
		res.PermissionIds = append(res.PermissionIds, permissionID)
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

func (s *Seeder) CreateRatelimit(ctx context.Context, req CreateRatelimitRequest) string {
	ratelimitID := uid.New(uid.RatelimitPrefix)
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
			CreatedAt:   time.Now().UnixMilli(),
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
			CreatedAt:   time.Now().UnixMilli(),
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
	metaBytes := []byte("{}")
	if len(req.Meta) > 0 {
		metaBytes = req.Meta
	}

	require.NoError(s.t, assert.NotEmpty(req.ExternalID, "Identity ExternalID must be set"))
	require.NoError(s.t, assert.NotEmpty(req.WorkspaceID, "Identity WorkspaceID must be set"))

	identityId := uid.New(uid.IdentityPrefix)
	err := db.Query.InsertIdentity(ctx, s.DB.RW(), db.InsertIdentityParams{
		ID:          identityId,
		ExternalID:  req.ExternalID,
		WorkspaceID: req.WorkspaceID,
		Environment: "",
		CreatedAt:   time.Now().UnixMilli(),
		Meta:        metaBytes,
	})
	require.NoError(s.t, err)

	for _, ratelimit := range req.Ratelimits {
		ratelimit.IdentityID = ptr.P(identityId)
		s.CreateRatelimit(ctx, ratelimit)
	}

	return identityId
}

type CreateRoleRequest struct {
	Name        string
	Description *string
	WorkspaceID string

	Permissions []CreatePermissionRequest
}

func (s *Seeder) CreateRole(ctx context.Context, req CreateRoleRequest) string {
	require.NoError(s.t, assert.NotEmpty(req.WorkspaceID, "Role WorkspaceID must be set"))
	require.NoError(s.t, assert.NotEmpty(req.Name, "Role Name must be set"))

	roleID := uid.New(uid.PermissionPrefix)

	err := db.Query.InsertRole(ctx, s.DB.RW(), db.InsertRoleParams{
		RoleID:      roleID,
		WorkspaceID: req.WorkspaceID,
		Name:        req.Name,
		CreatedAt:   time.Now().UnixMilli(),
		Description: sql.NullString{Valid: req.Description != nil, String: ptr.SafeDeref(req.Description, "")},
	})
	require.NoError(s.t, err)

	for _, permission := range req.Permissions {
		permissionID := s.CreatePermission(ctx, permission)
		err = db.Query.InsertRolePermission(ctx, s.DB.RW(), db.InsertRolePermissionParams{
			RoleID:       roleID,
			PermissionID: permissionID,
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

func (s *Seeder) CreatePermission(ctx context.Context, req CreatePermissionRequest) string {
	require.NoError(s.t, assert.NotEmpty(req.WorkspaceID, "Permission WorkspaceID must be set"))
	require.NoError(s.t, assert.NotEmpty(req.WorkspaceID, "Permission Name must be set"))
	require.NoError(s.t, assert.NotEmpty(req.WorkspaceID, "Permission Slug must be set"))

	permissionID := uid.New(uid.PermissionPrefix)
	err := db.Query.InsertPermission(ctx, s.DB.RW(), db.InsertPermissionParams{
		PermissionID: permissionID,
		WorkspaceID:  req.WorkspaceID,
		Name:         req.Name,
		Slug:         req.Slug,
		Description:  dbtype.NullString{Valid: req.Description != nil, String: ptr.SafeDeref(req.Description, "")},
		CreatedAtM:   time.Now().UnixMilli(),
	})
	require.NoError(s.t, err)

	return permissionID
}
