package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	mysql "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
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
	require.NoError(t, Query.InsertUserDeviceSession(ctx, database.RW(), InsertUserDeviceSessionParams{
		ID:               sessionID,
		WorkspaceID:      stored.WorkspaceID.String,
		UserID:           stored.ApproverUserID.String,
		Kind:             UserDeviceSessionsKindCli,
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
	rotated, err := Query.RotateUserDeviceSessionRefreshToken(ctx, database.RW(), RotateUserDeviceSessionRefreshTokenParams{
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

	session, err := Query.FindUserDeviceSessionByID(ctx, database.RO(), sessionID)
	require.NoError(t, err)
	requireJSONStrings(t, session.Permissions, []string{"api.*.read_api", "api.*.read_key"})
	require.Equal(t, sql.NullString{String: "Ada", Valid: true}, session.UserName)
	require.Equal(t, now+2*time.Hour.Milliseconds(), session.ExpiresAt)
}

// TestDeviceAuthorizationOAuthRoundTrip guarantees that the OAuth binding and
// the session's client and audience survive interpolated parameters, and that
// a CLI device login reads back without a binding.
func TestDeviceAuthorizationOAuthRoundTrip(t *testing.T) {
	skipWithoutDatabase(t)
	ctx := context.Background()
	database := openCLILoginDB(t, containers.MySQL(t))
	now := time.Now().UnixMilli()

	binding := mysqltype.DeviceAuthorizationOAuth{
		ClientID:      "https://client.example.com/oauth/metadata.json",
		RedirectURI:   "https://client.example.com/callback?tenant=a&b=it's",
		CodeChallenge: uid.Secure(43),
		State:         `st"ate`,
		Resource:      "https://mcp.example.com/mcp",
	}
	id := uid.New(uid.TestPrefix)
	require.NoError(t, Query.InsertDeviceAuthorization(ctx, database.RW(), InsertDeviceAuthorizationParams{
		ID:        id,
		UserCode:  uid.New(uid.TestPrefix, 20),
		ExpiresAt: now + time.Hour.Milliseconds(),
		Status:    DeviceAuthorizationsStatusPending,
		CreatedAt: now,
		Oauth:     &binding,
	}))
	stored, err := Query.FindDeviceAuthorizationByID(ctx, database.RO(), id)
	require.NoError(t, err)
	require.NotNil(t, stored.Oauth)
	require.Equal(t, binding, *stored.Oauth)

	device := insertPendingDeviceAuthorization(t, ctx, database, now+time.Hour.Milliseconds())
	require.Nil(t, device.Oauth)
	require.False(t, device.CodeHash.Valid)

	permissions, err := json.Marshal([]string{"api.*.read_api"})
	require.NoError(t, err)
	sessionID := uid.New(uid.TestPrefix)
	require.NoError(t, Query.InsertUserDeviceSession(ctx, database.RW(), InsertUserDeviceSessionParams{
		ID:               sessionID,
		WorkspaceID:      uid.New(uid.WorkspacePrefix),
		UserID:           uid.New(uid.TestPrefix),
		Kind:             UserDeviceSessionsKindMcp,
		Permissions:      permissions,
		RefreshTokenHash: uid.New(uid.TestPrefix, 32),
		RefreshFamily:    uid.New(uid.TestPrefix, 20),
		CreatedAt:        now,
		LastUsedAt:       now,
		ExpiresAt:        now + time.Hour.Milliseconds(),
		OauthClientID:    sql.NullString{String: binding.ClientID, Valid: true},
		Audience:         sql.NullString{String: binding.Resource, Valid: true},
	}))
	session, err := Query.FindUserDeviceSessionByID(ctx, database.RO(), sessionID)
	require.NoError(t, err)
	require.Equal(t, sql.NullString{String: binding.ClientID, Valid: true}, session.OauthClientID)
	require.Equal(t, sql.NullString{String: binding.Resource, Valid: true}, session.Audience)
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
	approvePendingAuthorization(t, ctx, database, consumed, now)
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
	reapproved := approvePendingAuthorization(t, ctx, database, consumed, now)
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
	require.Equal(t, int64(1), approvePendingAuthorization(t, ctx, database, approved, now))
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

// TestApprovedAuthorizationIsFoundByCodeHash guarantees that approval stores
// the authorization code hash and the shortened code lifetime, so the token
// endpoint can find the row by the code alone and an unknown code finds nothing.
func TestApprovedAuthorizationIsFoundByCodeHash(t *testing.T) {
	skipWithoutDatabase(t)
	ctx := context.Background()
	database := openCLILoginDB(t, containers.MySQL(t))
	now := time.Now().UnixMilli()

	row := insertPendingDeviceAuthorization(t, ctx, database, now+time.Hour.Milliseconds())
	codeHash := sql.NullString{String: hash.Sha256(uid.Secure(32)), Valid: true}
	codeExpiresAt := now + time.Minute.Milliseconds()
	roles, err := json.Marshal([]string{"admin"})
	require.NoError(t, err)
	permissions, err := json.Marshal([]string{"api.*.read_api"})
	require.NoError(t, err)
	approved, err := Query.ApproveDeviceAuthorization(ctx, database.RW(), ApproveDeviceAuthorizationParams{
		WorkspaceID:    sql.NullString{String: uid.New(uid.WorkspacePrefix), Valid: true},
		ApproverUserID: sql.NullString{String: uid.New(uid.TestPrefix), Valid: true},
		ApproverName:   sql.NullString{String: "", Valid: false},
		ApproverRoles:  roles,
		Permissions:    permissions,
		ApprovedAt:     sql.NullInt64{Int64: now, Valid: true},
		ExpiresAt:      codeExpiresAt,
		CodeHash:       codeHash,
		ID:             row.ID,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), approved)

	byCode, err := Query.FindDeviceAuthorizationByCodeHash(ctx, database.RW(), codeHash)
	require.NoError(t, err)
	require.Equal(t, row.ID, byCode.ID)
	require.Equal(t, DeviceAuthorizationsStatusApproved, byCode.Status)
	require.Equal(t, codeExpiresAt, byCode.ExpiresAt)

	_, err = Query.FindDeviceAuthorizationByCodeHash(ctx, database.RW(), sql.NullString{String: hash.Sha256(uid.Secure(32)), Valid: true})
	require.True(t, IsNotFound(err))
}

// TestListUserDeviceSessionsReturnsActiveRows guarantees that workspace and user
// lists omit revoked and idle-expired sessions.
func TestListUserDeviceSessionsReturnsActiveRows(t *testing.T) {
	skipWithoutDatabase(t)
	ctx := context.Background()
	database := openCLILoginDB(t, containers.MySQL(t))
	now := time.Now().UnixMilli()
	workspaceID := uid.New(uid.WorkspacePrefix)
	userID := uid.New(uid.TestPrefix)
	otherUserID := uid.New(uid.TestPrefix)

	active := insertActiveUserDeviceSession(t, ctx, database, userDeviceSessionFixture{
		WorkspaceID: workspaceID,
		UserID:      userID,
		CreatedAt:   now,
		ExpiresAt:   now + time.Hour.Milliseconds(),
	})
	revoked := insertActiveUserDeviceSession(t, ctx, database, userDeviceSessionFixture{
		WorkspaceID: workspaceID,
		UserID:      userID,
		CreatedAt:   now - time.Minute.Milliseconds(),
		ExpiresAt:   now + time.Hour.Milliseconds(),
	})
	idle := insertActiveUserDeviceSession(t, ctx, database, userDeviceSessionFixture{
		WorkspaceID: workspaceID,
		UserID:      userID,
		CreatedAt:   now - 2*time.Minute.Milliseconds(),
		ExpiresAt:   now - time.Minute.Milliseconds(),
	})
	other := insertActiveUserDeviceSession(t, ctx, database, userDeviceSessionFixture{
		WorkspaceID: workspaceID,
		UserID:      otherUserID,
		CreatedAt:   now - 3*time.Minute.Milliseconds(),
		ExpiresAt:   now + time.Hour.Milliseconds(),
	})

	revokedRows, err := Query.RevokeUserDeviceSession(ctx, database.RW(), RevokeUserDeviceSessionParams{
		RevokedAt:   sql.NullInt64{Int64: now, Valid: true},
		ID:          revoked.ID,
		WorkspaceID: workspaceID,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), revokedRows)

	byUser, err := Query.ListUserDeviceSessionsByUser(ctx, database.RO(), ListUserDeviceSessionsByUserParams{
		WorkspaceID: workspaceID,
		UserID:      userID,
		Now:         now,
	})
	require.NoError(t, err)
	require.Equal(t, []string{active.ID}, sessionIDs(byUser))

	byWorkspace, err := Query.ListUserDeviceSessionsByWorkspace(ctx, database.RO(), ListUserDeviceSessionsByWorkspaceParams{
		WorkspaceID: workspaceID,
		Now:         now,
	})
	require.NoError(t, err)
	require.Equal(t, []string{active.ID, other.ID}, sessionIDs(byWorkspace))
	require.NotContains(t, sessionIDs(byWorkspace), idle.ID)
	require.NotContains(t, sessionIDs(byWorkspace), revoked.ID)
}

// TestRevokeUserDeviceSessionFamilyReturnsAffectedRows guarantees that reuse
// detection can observe how many sessions in the family were revoked.
func TestRevokeUserDeviceSessionFamilyReturnsAffectedRows(t *testing.T) {
	skipWithoutDatabase(t)
	ctx := context.Background()
	database := openCLILoginDB(t, containers.MySQL(t))
	now := time.Now().UnixMilli()
	workspaceID := uid.New(uid.WorkspacePrefix)
	family := uid.New(uid.TestPrefix, 20)

	insertActiveUserDeviceSession(t, ctx, database, userDeviceSessionFixture{
		WorkspaceID:   workspaceID,
		UserID:        uid.New(uid.TestPrefix),
		RefreshFamily: family,
		CreatedAt:     now,
		ExpiresAt:     now + time.Hour.Milliseconds(),
	})
	insertActiveUserDeviceSession(t, ctx, database, userDeviceSessionFixture{
		WorkspaceID:   workspaceID,
		UserID:        uid.New(uid.TestPrefix),
		RefreshFamily: family,
		CreatedAt:     now,
		ExpiresAt:     now + time.Hour.Milliseconds(),
	})
	insertActiveUserDeviceSession(t, ctx, database, userDeviceSessionFixture{
		WorkspaceID: workspaceID,
		UserID:      uid.New(uid.TestPrefix),
		CreatedAt:   now,
		ExpiresAt:   now + time.Hour.Milliseconds(),
	})

	revoked, err := Query.RevokeUserDeviceSessionFamily(ctx, database.RW(), RevokeUserDeviceSessionFamilyParams{
		RevokedAt:     sql.NullInt64{Int64: now, Valid: true},
		RefreshFamily: family,
	})
	require.NoError(t, err)
	require.Equal(t, int64(2), revoked)

	again, err := Query.RevokeUserDeviceSessionFamily(ctx, database.RW(), RevokeUserDeviceSessionFamilyParams{
		RevokedAt:     sql.NullInt64{Int64: now, Valid: true},
		RefreshFamily: family,
	})
	require.NoError(t, err)
	require.Equal(t, int64(0), again)
}

// TestFindUnrevokedUserDeviceSessionByRefreshFamily guarantees that refresh reuse
// detection finds a live session by family and ignores a revoked one.
func TestFindUnrevokedUserDeviceSessionByRefreshFamily(t *testing.T) {
	skipWithoutDatabase(t)
	ctx := context.Background()
	database := openCLILoginDB(t, containers.MySQL(t))
	now := time.Now().UnixMilli()
	family := uid.New(uid.TestPrefix, 20)

	session := insertActiveUserDeviceSession(t, ctx, database, userDeviceSessionFixture{
		WorkspaceID:   uid.New(uid.WorkspacePrefix),
		UserID:        uid.New(uid.TestPrefix),
		RefreshFamily: family,
		CreatedAt:     now,
		ExpiresAt:     now + time.Hour.Milliseconds(),
	})

	found, err := Query.FindUnrevokedUserDeviceSessionByRefreshFamily(ctx, database.RW(), family)
	require.NoError(t, err)
	require.Equal(t, session.ID, found.ID)

	_, err = Query.FindUnrevokedUserDeviceSessionByRefreshFamily(ctx, database.RW(), uid.New(uid.TestPrefix, 20))
	require.True(t, IsNotFound(err))

	revoked, err := Query.RevokeUserDeviceSessionFamily(ctx, database.RW(), RevokeUserDeviceSessionFamilyParams{
		RevokedAt:     sql.NullInt64{Int64: now, Valid: true},
		RefreshFamily: family,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), revoked)

	_, err = Query.FindUnrevokedUserDeviceSessionByRefreshFamily(ctx, database.RW(), family)
	require.True(t, IsNotFound(err))
}

// TestDeleteExpiredUserDeviceSessionsRemovesOnlyExpiredRows guarantees that idle
// sessions past the cutoff are deleted and a session that is still live is
// left in place.
func TestDeleteExpiredUserDeviceSessionsRemovesOnlyExpiredRows(t *testing.T) {
	skipWithoutDatabase(t)
	ctx := context.Background()
	database := openCLILoginDB(t, containers.MySQLLogicalIsolated(t))
	now := time.Now().UnixMilli()
	workspaceID := uid.New(uid.WorkspacePrefix)

	expired := insertActiveUserDeviceSession(t, ctx, database, userDeviceSessionFixture{
		WorkspaceID: workspaceID,
		UserID:      uid.New(uid.TestPrefix),
		CreatedAt:   now - time.Hour.Milliseconds(),
		ExpiresAt:   now - time.Minute.Milliseconds(),
	})
	live := insertActiveUserDeviceSession(t, ctx, database, userDeviceSessionFixture{
		WorkspaceID: workspaceID,
		UserID:      uid.New(uid.TestPrefix),
		CreatedAt:   now,
		ExpiresAt:   now + time.Hour.Milliseconds(),
	})

	deleted, err := Query.DeleteExpiredUserDeviceSessions(ctx, database.RW(), now)
	require.NoError(t, err)
	require.Equal(t, int64(1), deleted)

	_, err = Query.FindUserDeviceSessionByID(ctx, database.RW(), expired.ID)
	require.True(t, IsNotFound(err))
	_, err = Query.FindUserDeviceSessionByID(ctx, database.RW(), live.ID)
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

func approvePendingAuthorization(t *testing.T, ctx context.Context, database *database, row DeviceAuthorization, approvedAt int64) int64 {
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
		ExpiresAt:      row.ExpiresAt,
		CodeHash:       sql.NullString{String: "", Valid: false},
		ID:             row.ID,
	})
	require.NoError(t, err)
	return n
}

type userDeviceSessionFixture struct {
	WorkspaceID   string
	UserID        string
	RefreshFamily string
	CreatedAt     int64
	ExpiresAt     int64
}

func insertActiveUserDeviceSession(t *testing.T, ctx context.Context, database *database, fixture userDeviceSessionFixture) UserDeviceSession {
	t.Helper()
	permissions, err := json.Marshal([]string{"api.*.read_api"})
	require.NoError(t, err)
	family := fixture.RefreshFamily
	if family == "" {
		family = uid.New(uid.TestPrefix, 20)
	}
	id := uid.New(uid.TestPrefix)
	require.NoError(t, Query.InsertUserDeviceSession(ctx, database.RW(), InsertUserDeviceSessionParams{
		ID:               id,
		WorkspaceID:      fixture.WorkspaceID,
		UserID:           fixture.UserID,
		Kind:             UserDeviceSessionsKindCli,
		Permissions:      permissions,
		DeviceName:       sql.NullString{String: "laptop", Valid: true},
		RefreshTokenHash: uid.New(uid.TestPrefix, 32),
		RefreshFamily:    family,
		CreatedAt:        fixture.CreatedAt,
		LastUsedAt:       fixture.CreatedAt,
		ExpiresAt:        fixture.ExpiresAt,
	}))
	row, err := Query.FindUserDeviceSessionByID(ctx, database.RW(), id)
	require.NoError(t, err)
	return row
}

func requireJSONStrings(t *testing.T, raw []byte, want []string) {
	t.Helper()
	var got []string
	require.NoError(t, json.Unmarshal(raw, &got))
	require.Equal(t, want, got)
}

func sessionIDs(rows []UserDeviceSession) []string {
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	return ids
}
