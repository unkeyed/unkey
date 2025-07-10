package seed

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/hash"
	"github.com/unkeyed/unkey/go/pkg/uid"
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
	Resources Resources
}

// New creates a new Seeder instance
func New(t *testing.T, database db.Database) *Seeder {
	return &Seeder{
		t:         t,
		DB:        database,
		Resources: Resources{}, //nolint:exhaustruct
	}
}

func (s *Seeder) CreateWorkspace(ctx context.Context) db.Workspace {
	params := db.InsertWorkspaceParams{
		ID:        uid.New("test_ws"),
		OrgID:     uid.New("test_org"),
		Name:      uid.New("test_name"),
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
	s.Resources.RootApi = s.CreateAPI(ctx, s.Resources.RootWorkspace.ID, false)
	keyring, err := db.Query.FindKeyringByID(ctx, s.DB.RW(), s.Resources.RootApi.KeyAuthID.String)
	require.NoError(s.t, err)
	s.Resources.RootKeyring = keyring
}

func (s *Seeder) CreateAPI(ctx context.Context, workspaceID string, encryptedKeys bool) db.Api {
	keyAuthID := uid.New(uid.KeyAuthPrefix)
	err := db.Query.InsertKeyring(ctx, s.DB.RW(), db.InsertKeyringParams{
		ID:                 keyAuthID,
		WorkspaceID:        workspaceID,
		CreatedAtM:         time.Now().UnixMilli(),
		DefaultPrefix:      sql.NullString{Valid: false},
		DefaultBytes:       sql.NullInt32{Valid: false},
		StoreEncryptedKeys: encryptedKeys,
	})
	require.NoError(s.t, err)

	apiID := uid.New("api")
	err = db.Query.InsertApi(ctx, s.DB.RW(), db.InsertApiParams{
		ID:          apiID,
		Name:        "test-api-db",
		WorkspaceID: workspaceID,
		AuthType:    db.NullApisAuthType{Valid: true, ApisAuthType: db.ApisAuthTypeKey},
		KeyAuthID:   sql.NullString{Valid: true, String: keyAuthID},
		CreatedAtM:  time.Now().UnixMilli(),
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
				Description:  sql.NullString{String: "", Valid: false},
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
	WorkspaceID, KeyAuthID string
	Expires                *time.Time
	Disabled               bool
}

type CreateKeyResponse struct {
	KeyID, Key string
}

func (s *Seeder) CreateKey(ctx context.Context, req CreateKeyRequest) CreateKeyResponse {
	keyID := uid.New(uid.KeyPrefix)
	key := uid.New("")
	start := key[:4]

	expires := time.Now()
	if req.Expires != nil {
		expires = *req.Expires
	}

	err := db.Query.InsertKey(ctx, s.DB.RW(), db.InsertKeyParams{
		ID:                keyID,
		KeyringID:         req.KeyAuthID,
		WorkspaceID:       req.WorkspaceID,
		CreatedAtM:        time.Now().UnixMilli(),
		Hash:              hash.Sha256(key),
		Enabled:           !req.Disabled,
		Start:             start,
		Name:              sql.NullString{String: "test-key", Valid: true},
		ForWorkspaceID:    sql.NullString{String: "", Valid: false},
		IdentityID:        sql.NullString{String: "", Valid: false},
		Meta:              sql.NullString{String: "", Valid: false},
		Expires:           sql.NullTime{Time: expires, Valid: req.Expires != nil},
		RemainingRequests: sql.NullInt32{Int32: 0, Valid: false},
		RefillAmount:      sql.NullInt32{Int32: 0, Valid: false},
		RefillDay:         sql.NullInt16{Int16: 0, Valid: false},
	})

	require.NoError(s.t, err)

	return CreateKeyResponse{
		KeyID: keyID,
		Key:   key,
	}
}
