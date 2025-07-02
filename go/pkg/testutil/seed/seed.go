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
	// Insert root workspace

	s.Resources.RootWorkspace = s.CreateWorkspace(ctx)

	// Insert root keyring
	insertRootKeyringParams := db.InsertKeyringParams{
		ID:                 uid.New("test_kr"),
		WorkspaceID:        s.Resources.RootWorkspace.ID,
		StoreEncryptedKeys: false,
		DefaultPrefix:      sql.NullString{String: "test", Valid: true},
		DefaultBytes:       sql.NullInt32{Int32: 8, Valid: true},
		CreatedAtM:         time.Now().UnixMilli(),
	}

	err := db.Query.InsertKeyring(ctx, s.DB.RW(), insertRootKeyringParams)
	require.NoError(s.t, err)

	s.Resources.RootKeyring, err = db.Query.FindKeyringByID(ctx, s.DB.RW(), insertRootKeyringParams.ID)
	require.NoError(s.t, err)

	s.Resources.UserWorkspace = s.CreateWorkspace(ctx)

	require.NoError(s.t, err)
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
		RatelimitAsync:    sql.NullBool{Bool: false, Valid: false},
		RatelimitLimit:    sql.NullInt32{Int32: 0, Valid: false},
		RatelimitDuration: sql.NullInt64{Int64: 0, Valid: false},
		Environment:       sql.NullString{String: "", Valid: false},
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
				// Error 1062 (23000): Duplicate entry
				require.Equal(s.t, uint16(1062), mysqlErr.Number, "Unexpected MySQL error number, got %d, expected %d", mysqlErr.Number, uint16(1062))
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
