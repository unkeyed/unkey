package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	mysql "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

// TestDeviceAuthorizationJSONRoundTrip guarantees that approver roles and
// permissions written with CAST(? AS JSON) round-trip when the driver
// interpolates parameters, which is how production connects to Vitess.
func TestDeviceAuthorizationJSONRoundTrip(t *testing.T) {
	skipWithoutDatabase(t)
	ctx := context.Background()
	database := openCLILoginDB(t, containers.MySQL(t))
	now := time.Now().UnixMilli()

	roles, err := json.Marshal([]string{"admin", "developer"})
	require.NoError(t, err)
	permissions, err := json.Marshal([]string{"api.*.read_api", "identity.*.read_identity"})
	require.NoError(t, err)

	row := insertPendingDeviceAuthorization(t, ctx, database, now+time.Hour.Milliseconds())
	approved, err := Query.ApproveDeviceAuthorization(ctx, database.RW(), ApproveDeviceAuthorizationParams{
		WorkspaceID:    sql.NullString{String: uid.New(uid.WorkspacePrefix), Valid: true},
		ApproverUserID: sql.NullString{String: uid.New(uid.TestPrefix), Valid: true},
		ApproverName:   sql.NullString{String: "Approver", Valid: true},
		ApproverRoles:  roles,
		Permissions:    permissions,
		ApprovedAt:     sql.NullInt64{Int64: now, Valid: true},
		ID:             row.ID,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), approved)

	stored, err := Query.FindDeviceAuthorizationByID(ctx, database.RO(), row.ID)
	require.NoError(t, err)
	require.Equal(t, DeviceAuthorizationsStatusApproved, stored.Status)
	requireJSONStrings(t, stored.ApproverRoles, []string{"admin", "developer"})
	requireJSONStrings(t, stored.Permissions, []string{"api.*.read_api", "identity.*.read_identity"})

	sessionPermissions, err := json.Marshal([]string{"api.*.read_api"})
	require.NoError(t, err)
	sessionID := uid.New(uid.TestPrefix)
	refreshHash := uid.New(uid.TestPrefix, 32)
	require.NoError(t, Query.InsertDeviceSession(ctx, database.RW(), InsertDeviceSessionParams{
		ID:               sessionID,
		Kind:             DeviceSessionsKindCli,
		WorkspaceID:      stored.WorkspaceID.String,
		UserID:           stored.ApproverUserID.String,
		UserName:         sql.NullString{String: "Ada", Valid: true},
		Permissions:      sessionPermissions,
		DeviceName:       sql.NullString{String: "laptop", Valid: true},
		RefreshTokenHash: refreshHash,
		RefreshFamily:    uid.New(uid.TestPrefix, 20),
		CreatedAt:        now,
		LastUsedAt:       now,
		ExpiresAt:        now + time.Hour.Milliseconds(),
	}))

	rotatedPermissions, err := json.Marshal([]string{"api.*.read_api", "api.*.read_key"})
	require.NoError(t, err)
	nextHash := uid.New(uid.TestPrefix, 32)
	rotated, err := Query.RotateDeviceSessionRefreshToken(ctx, database.RW(), RotateDeviceSessionRefreshTokenParams{
		RefreshTokenHash:         nextHash,
		PreviousRefreshTokenHash: sql.NullString{String: refreshHash, Valid: true},
		RotatedAt:                sql.NullInt64{Int64: now, Valid: true},
		LastUsedAt:               now,
		ExpiresAt:                now + 2*time.Hour.Milliseconds(),
		Permissions:              rotatedPermissions,
		ID:                       sessionID,
		CurrentRefreshTokenHash:  refreshHash,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), rotated)

	session, err := Query.FindDeviceSessionByID(ctx, database.RO(), sessionID)
	require.NoError(t, err)
	requireJSONStrings(t, session.Permissions, []string{"api.*.read_api", "api.*.read_key"})
	require.Equal(t, sql.NullString{String: "Ada", Valid: true}, session.UserName)
	require.Equal(t, now+2*time.Hour.Milliseconds(), session.ExpiresAt)
}

// TestDeviceAuthorizationStatusTransitions guarantees that a consumed login
// cannot be denied, expired, or approved again. Reopening it would let a
// second poll consume the same authorization.
func TestDeviceAuthorizationStatusTransitions(t *testing.T) {
	skipWithoutDatabase(t)
	ctx := context.Background()
	database := openCLILoginDB(t, containers.MySQL(t))
	now := time.Now().UnixMilli()

	consumed := insertPendingDeviceAuthorization(t, ctx, database, now+time.Hour.Milliseconds())
	approvePendingAuthorization(t, ctx, database, consumed.ID, now)
	n, err := Query.ConsumeDeviceAuthorization(ctx, database.RW(), ConsumeDeviceAuthorizationParams{
		SessionID: sql.NullString{String: uid.New(uid.TestPrefix), Valid: true},
		ID:        consumed.ID,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), n)

	denied, err := Query.DenyDeviceAuthorization(ctx, database.RW(), consumed.ID)
	require.NoError(t, err)
	require.Equal(t, int64(0), denied)
	expired, err := Query.ExpireDeviceAuthorization(ctx, database.RW(), consumed.ID)
	require.NoError(t, err)
	require.Equal(t, int64(0), expired)
	reapproved := approvePendingAuthorization(t, ctx, database, consumed.ID, now)
	require.Equal(t, int64(0), reapproved)

	stored, err := Query.FindDeviceAuthorizationByID(ctx, database.RW(), consumed.ID)
	require.NoError(t, err)
	require.Equal(t, DeviceAuthorizationsStatusConsumed, stored.Status)

	pending := insertPendingDeviceAuthorization(t, ctx, database, now+time.Hour.Milliseconds())
	denied, err = Query.DenyDeviceAuthorization(ctx, database.RW(), pending.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1), denied)
	deniedAgain, err := Query.DenyDeviceAuthorization(ctx, database.RW(), pending.ID)
	require.NoError(t, err)
	require.Equal(t, int64(0), deniedAgain)

	approved := insertPendingDeviceAuthorization(t, ctx, database, now+time.Hour.Milliseconds())
	require.Equal(t, int64(1), approvePendingAuthorization(t, ctx, database, approved.ID, now))
	denied, err = Query.DenyDeviceAuthorization(ctx, database.RW(), approved.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1), denied)

	expiring := insertPendingDeviceAuthorization(t, ctx, database, now-time.Minute.Milliseconds())
	expired, err = Query.ExpireDeviceAuthorization(ctx, database.RW(), expiring.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1), expired)
	stored, err = Query.FindDeviceAuthorizationByID(ctx, database.RW(), expiring.ID)
	require.NoError(t, err)
	require.Equal(t, DeviceAuthorizationsStatusExpired, stored.Status)
}

// TestListDeviceSessionsReturnsActiveRows guarantees that workspace and user
// lists omit revoked and idle-expired sessions.
func TestListDeviceSessionsReturnsActiveRows(t *testing.T) {
	skipWithoutDatabase(t)
	ctx := context.Background()
	database := openCLILoginDB(t, containers.MySQL(t))
	now := time.Now().UnixMilli()
	workspaceID := uid.New(uid.WorkspacePrefix)
	userID := uid.New(uid.TestPrefix)
	otherUserID := uid.New(uid.TestPrefix)

	active := insertActiveDeviceSession(t, ctx, database, deviceSessionFixture{
		WorkspaceID: workspaceID,
		UserID:      userID,
		CreatedAt:   now,
		ExpiresAt:   now + time.Hour.Milliseconds(),
	})
	revoked := insertActiveDeviceSession(t, ctx, database, deviceSessionFixture{
		WorkspaceID: workspaceID,
		UserID:      userID,
		CreatedAt:   now - time.Minute.Milliseconds(),
		ExpiresAt:   now + time.Hour.Milliseconds(),
	})
	idle := insertActiveDeviceSession(t, ctx, database, deviceSessionFixture{
		WorkspaceID: workspaceID,
		UserID:      userID,
		CreatedAt:   now - 2*time.Minute.Milliseconds(),
		ExpiresAt:   now - time.Minute.Milliseconds(),
	})
	other := insertActiveDeviceSession(t, ctx, database, deviceSessionFixture{
		WorkspaceID: workspaceID,
		UserID:      otherUserID,
		CreatedAt:   now - 3*time.Minute.Milliseconds(),
		ExpiresAt:   now + time.Hour.Milliseconds(),
	})

	revokedRows, err := Query.RevokeDeviceSession(ctx, database.RW(), RevokeDeviceSessionParams{
		RevokedAt:   sql.NullInt64{Int64: now, Valid: true},
		ID:          revoked.ID,
		WorkspaceID: workspaceID,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), revokedRows)

	byUser, err := Query.ListDeviceSessionsByUser(ctx, database.RO(), ListDeviceSessionsByUserParams{
		WorkspaceID: workspaceID,
		UserID:      userID,
		Now:         now,
	})
	require.NoError(t, err)
	require.Equal(t, []string{active.ID}, sessionIDs(byUser))

	byWorkspace, err := Query.ListDeviceSessionsByWorkspace(ctx, database.RO(), ListDeviceSessionsByWorkspaceParams{
		WorkspaceID: workspaceID,
		Now:         now,
	})
	require.NoError(t, err)
	require.Equal(t, []string{active.ID, other.ID}, sessionIDs(byWorkspace))
	require.NotContains(t, sessionIDs(byWorkspace), idle.ID)
	require.NotContains(t, sessionIDs(byWorkspace), revoked.ID)
}

// TestRevokeDeviceSessionFamilyReturnsAffectedRows guarantees that reuse
// detection can observe how many sessions in the family were revoked.
func TestRevokeDeviceSessionFamilyReturnsAffectedRows(t *testing.T) {
	skipWithoutDatabase(t)
	ctx := context.Background()
	database := openCLILoginDB(t, containers.MySQL(t))
	now := time.Now().UnixMilli()
	workspaceID := uid.New(uid.WorkspacePrefix)
	family := uid.New(uid.TestPrefix, 20)

	insertActiveDeviceSession(t, ctx, database, deviceSessionFixture{
		WorkspaceID:   workspaceID,
		UserID:        uid.New(uid.TestPrefix),
		RefreshFamily: family,
		CreatedAt:     now,
		ExpiresAt:     now + time.Hour.Milliseconds(),
	})
	insertActiveDeviceSession(t, ctx, database, deviceSessionFixture{
		WorkspaceID:   workspaceID,
		UserID:        uid.New(uid.TestPrefix),
		RefreshFamily: family,
		CreatedAt:     now,
		ExpiresAt:     now + time.Hour.Milliseconds(),
	})
	insertActiveDeviceSession(t, ctx, database, deviceSessionFixture{
		WorkspaceID: workspaceID,
		UserID:      uid.New(uid.TestPrefix),
		CreatedAt:   now,
		ExpiresAt:   now + time.Hour.Milliseconds(),
	})

	revoked, err := Query.RevokeDeviceSessionFamily(ctx, database.RW(), RevokeDeviceSessionFamilyParams{
		RevokedAt:     sql.NullInt64{Int64: now, Valid: true},
		RefreshFamily: family,
	})
	require.NoError(t, err)
	require.Equal(t, int64(2), revoked)

	again, err := Query.RevokeDeviceSessionFamily(ctx, database.RW(), RevokeDeviceSessionFamilyParams{
		RevokedAt:     sql.NullInt64{Int64: now, Valid: true},
		RefreshFamily: family,
	})
	require.NoError(t, err)
	require.Equal(t, int64(0), again)
}

// TestFindUnrevokedDeviceSessionByRefreshFamily guarantees that refresh reuse
// detection finds a live session by family and ignores a revoked one.
func TestFindUnrevokedDeviceSessionByRefreshFamily(t *testing.T) {
	skipWithoutDatabase(t)
	ctx := context.Background()
	database := openCLILoginDB(t, containers.MySQL(t))
	now := time.Now().UnixMilli()
	family := uid.New(uid.TestPrefix, 20)

	session := insertActiveDeviceSession(t, ctx, database, deviceSessionFixture{
		WorkspaceID:   uid.New(uid.WorkspacePrefix),
		UserID:        uid.New(uid.TestPrefix),
		RefreshFamily: family,
		CreatedAt:     now,
		ExpiresAt:     now + time.Hour.Milliseconds(),
	})

	found, err := Query.FindUnrevokedDeviceSessionByRefreshFamily(ctx, database.RW(), family)
	require.NoError(t, err)
	require.Equal(t, session.ID, found.ID)

	_, err = Query.FindUnrevokedDeviceSessionByRefreshFamily(ctx, database.RW(), uid.New(uid.TestPrefix, 20))
	require.True(t, IsNotFound(err))

	revoked, err := Query.RevokeDeviceSessionFamily(ctx, database.RW(), RevokeDeviceSessionFamilyParams{
		RevokedAt:     sql.NullInt64{Int64: now, Valid: true},
		RefreshFamily: family,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), revoked)

	_, err = Query.FindUnrevokedDeviceSessionByRefreshFamily(ctx, database.RW(), family)
	require.True(t, IsNotFound(err))
}

// TestDeleteExpiredDeviceSessionsRemovesOnlyExpiredRows guarantees that idle
// sessions past the cutoff are deleted and a session that is still live is
// left in place.
func TestDeleteExpiredDeviceSessionsRemovesOnlyExpiredRows(t *testing.T) {
	skipWithoutDatabase(t)
	ctx := context.Background()
	database := openCLILoginDB(t, containers.MySQLLogicalIsolated(t))
	now := time.Now().UnixMilli()
	workspaceID := uid.New(uid.WorkspacePrefix)

	expired := insertActiveDeviceSession(t, ctx, database, deviceSessionFixture{
		WorkspaceID: workspaceID,
		UserID:      uid.New(uid.TestPrefix),
		CreatedAt:   now - time.Hour.Milliseconds(),
		ExpiresAt:   now - time.Minute.Milliseconds(),
	})
	live := insertActiveDeviceSession(t, ctx, database, deviceSessionFixture{
		WorkspaceID: workspaceID,
		UserID:      uid.New(uid.TestPrefix),
		CreatedAt:   now,
		ExpiresAt:   now + time.Hour.Milliseconds(),
	})

	deleted, err := Query.DeleteExpiredDeviceSessions(ctx, database.RW(), now)
	require.NoError(t, err)
	require.Equal(t, int64(1), deleted)

	_, err = Query.FindDeviceSessionByID(ctx, database.RW(), expired.ID)
	require.True(t, IsNotFound(err))
	_, err = Query.FindDeviceSessionByID(ctx, database.RW(), live.ID)
	require.NoError(t, err)
}

func skipWithoutDatabase(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
}

func openCLILoginDB(t *testing.T, mysqlCfg containers.MySQLConfig) *database {
	t.Helper()

	dsn, err := mysql.ParseDSN(mysqlCfg.DSN)
	require.NoError(t, err)
	dsn.InterpolateParams = true

	database, err := New(Config{
		PrimaryDSN:  dsn.FormatDSN(),
		ReadOnlyDSN: "",
		Tags:        sqlcomment.Disabled(),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, database.Close())
	})
	return database
}

func insertPendingDeviceAuthorization(t *testing.T, ctx context.Context, database *database, expiresAt int64) DeviceAuthorization {
	t.Helper()
	id := uid.New(uid.TestPrefix)
	require.NoError(t, Query.InsertDeviceAuthorization(ctx, database.RW(), InsertDeviceAuthorizationParams{
		ID:        id,
		UserCode:  uid.New(uid.TestPrefix, 20),
		ExpiresAt: expiresAt,
		Status:    DeviceAuthorizationsStatusPending,
		CreatedAt: time.Now().UnixMilli(),
	}))
	row, err := Query.FindDeviceAuthorizationByID(ctx, database.RW(), id)
	require.NoError(t, err)
	return row
}

func approvePendingAuthorization(t *testing.T, ctx context.Context, database *database, id string, approvedAt int64) int64 {
	t.Helper()
	roles, err := json.Marshal([]string{"developer"})
	require.NoError(t, err)
	permissions, err := json.Marshal([]string{"api.*.read_api"})
	require.NoError(t, err)
	n, err := Query.ApproveDeviceAuthorization(ctx, database.RW(), ApproveDeviceAuthorizationParams{
		WorkspaceID:    sql.NullString{String: uid.New(uid.WorkspacePrefix), Valid: true},
		ApproverUserID: sql.NullString{String: uid.New(uid.TestPrefix), Valid: true},
		ApproverName:   sql.NullString{String: "Approver", Valid: true},
		ApproverRoles:  roles,
		Permissions:    permissions,
		ApprovedAt:     sql.NullInt64{Int64: approvedAt, Valid: true},
		ID:             id,
	})
	require.NoError(t, err)
	return n
}

type deviceSessionFixture struct {
	WorkspaceID   string
	UserID        string
	RefreshFamily string
	CreatedAt     int64
	ExpiresAt     int64
}

func insertActiveDeviceSession(t *testing.T, ctx context.Context, database *database, fixture deviceSessionFixture) DeviceSession {
	t.Helper()
	permissions, err := json.Marshal([]string{"api.*.read_api"})
	require.NoError(t, err)
	family := fixture.RefreshFamily
	if family == "" {
		family = uid.New(uid.TestPrefix, 20)
	}
	id := uid.New(uid.TestPrefix)
	require.NoError(t, Query.InsertDeviceSession(ctx, database.RW(), InsertDeviceSessionParams{
		ID:               id,
		Kind:             DeviceSessionsKindCli,
		WorkspaceID:      fixture.WorkspaceID,
		UserID:           fixture.UserID,
		Permissions:      permissions,
		DeviceName:       sql.NullString{String: "laptop", Valid: true},
		RefreshTokenHash: uid.New(uid.TestPrefix, 32),
		RefreshFamily:    family,
		CreatedAt:        fixture.CreatedAt,
		LastUsedAt:       fixture.CreatedAt,
		ExpiresAt:        fixture.ExpiresAt,
	}))
	row, err := Query.FindDeviceSessionByID(ctx, database.RW(), id)
	require.NoError(t, err)
	return row
}

func requireJSONStrings(t *testing.T, raw []byte, want []string) {
	t.Helper()
	var got []string
	require.NoError(t, json.Unmarshal(raw, &got))
	require.Equal(t, want, got)
}

func sessionIDs(rows []DeviceSession) []string {
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	return ids
}
