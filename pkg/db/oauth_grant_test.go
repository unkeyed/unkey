package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

// TestRedirectAuthorizationRoundTrip guarantees that an OAuth authorization
// request keeps every redirect column it was created with, and that approval
// stores the authorization code hash and the shortened code lifetime so the
// token endpoint can find the row by the code alone.
func TestRedirectAuthorizationRoundTrip(t *testing.T) {
	skipWithoutDatabase(t)
	ctx := context.Background()
	database := openCLILoginDB(t, containers.MySQL(t))
	now := time.Now().UnixMilli()

	id := uid.New(uid.DeviceAuthorizationPrefix)
	clientID := uid.New(uid.OAuthClientPrefix)
	redirectURI := "https://gateway.test/oauth/callback?tenant=" + uid.New(uid.TestPrefix)
	state := uid.New(uid.TestPrefix, 32)
	challenge := uid.New(uid.TestPrefix, 43)
	require.NoError(t, Query.InsertRedirectAuthorization(ctx, database.RW(), InsertRedirectAuthorizationParams{
		ID:                   id,
		UserCode:             uid.New(uid.TestPrefix, 26),
		ExpiresAt:            now + time.Hour.Milliseconds(),
		DeviceName:           sql.NullString{String: "Unkey MCP", Valid: true},
		RequesterIp:          sql.NullString{String: "203.0.113.10", Valid: true},
		RequesterUserAgent:   sql.NullString{},
		ClientID:             sql.NullString{String: clientID, Valid: true},
		RedirectUri:          sql.NullString{String: redirectURI, Valid: true},
		RedirectUriPresented: sql.NullBool{Bool: false, Valid: true},
		State:                sql.NullString{String: state, Valid: true},
		CodeChallenge:        sql.NullString{String: challenge, Valid: true},
		CreatedAt:            now,
	}))

	pending, err := Query.FindDeviceAuthorizationByID(ctx, database.RW(), id)
	require.NoError(t, err)
	require.Equal(t, DeviceAuthorizationsStatusPending, pending.Status)
	require.Equal(t, sql.NullString{String: clientID, Valid: true}, pending.ClientID)
	require.Equal(t, sql.NullString{String: redirectURI, Valid: true}, pending.RedirectUri)
	require.Equal(t, sql.NullBool{Bool: false, Valid: true}, pending.RedirectUriPresented)
	require.Equal(t, sql.NullString{String: state, Valid: true}, pending.State)
	require.Equal(t, sql.NullString{String: challenge, Valid: true}, pending.CodeChallenge)
	require.False(t, pending.CodeHash.Valid)

	codeHash := hash.Sha256(uid.New(uid.TestPrefix, 32))
	codeExpiresAt := now + time.Minute.Milliseconds()
	roles, err := json.Marshal([]string{"admin"})
	require.NoError(t, err)
	permissions, err := json.Marshal([]string{"api.*.read_api"})
	require.NoError(t, err)
	approved, err := Query.ApproveDeviceAuthorization(ctx, database.RW(), ApproveDeviceAuthorizationParams{
		WorkspaceID:    sql.NullString{String: uid.New(uid.WorkspacePrefix), Valid: true},
		ApproverUserID: sql.NullString{String: uid.New(uid.TestPrefix), Valid: true},
		ApproverName:   sql.NullString{},
		ApproverRoles:  roles,
		Permissions:    permissions,
		ApprovedAt:     sql.NullInt64{Int64: now, Valid: true},
		ExpiresAt:      codeExpiresAt,
		CodeHash:       sql.NullString{String: codeHash, Valid: true},
		ID:             id,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), approved)

	byCode, err := Query.FindDeviceAuthorizationByCodeHash(ctx, database.RW(), sql.NullString{String: codeHash, Valid: true})
	require.NoError(t, err)
	require.Equal(t, id, byCode.ID)
	require.Equal(t, DeviceAuthorizationsStatusApproved, byCode.Status)
	require.Equal(t, codeExpiresAt, byCode.ExpiresAt)

	_, err = Query.FindDeviceAuthorizationByCodeHash(ctx, database.RW(), sql.NullString{String: hash.Sha256(uid.New(uid.TestPrefix, 32)), Valid: true})
	require.True(t, IsNotFound(err))
}

// TestDeviceAuthorizationLeavesRedirectColumnsNull guarantees that a CLI
// device login cannot carry OAuth redirect state: the grant service decides
// the delivery mode from client_id alone.
func TestDeviceAuthorizationLeavesRedirectColumnsNull(t *testing.T) {
	skipWithoutDatabase(t)
	ctx := context.Background()
	database := openCLILoginDB(t, containers.MySQL(t))

	row := insertPendingDeviceAuthorization(t, ctx, database, time.Now().Add(time.Hour).UnixMilli())
	require.False(t, row.ClientID.Valid)
	require.False(t, row.RedirectUri.Valid)
	require.False(t, row.RedirectUriPresented.Valid)
	require.False(t, row.State.Valid)
	require.False(t, row.CodeChallenge.Valid)
	require.False(t, row.CodeHash.Valid)
}

// TestOAuthClientMetadataHashIsUnique guarantees that registering the same
// client metadata twice cannot add a second row, and that the existing client
// can be found by that hash so registration can return it instead.
func TestOAuthClientMetadataHashIsUnique(t *testing.T) {
	skipWithoutDatabase(t)
	ctx := context.Background()
	database := openCLILoginDB(t, containers.MySQL(t))
	now := time.Now().UnixMilli()

	redirectURIs, err := json.Marshal([]string{"https://gateway.test/oauth/callback"})
	require.NoError(t, err)
	metadataHash := hash.Sha256(uid.New(uid.TestPrefix, 32))
	first := InsertOAuthClientParams{
		ID:                 uid.New(uid.OAuthClientPrefix, 16),
		Name:               "Unkey MCP",
		RedirectUris:       redirectURIs,
		MetadataHash:       metadataHash,
		RegistrarIp:        sql.NullString{String: "203.0.113.20", Valid: true},
		RegistrarUserAgent: sql.NullString{},
		CreatedAt:          now,
	}
	require.NoError(t, Query.InsertOAuthClient(ctx, database.RW(), first))

	duplicate := first
	duplicate.ID = uid.New(uid.OAuthClientPrefix, 16)
	err = Query.InsertOAuthClient(ctx, database.RW(), duplicate)
	require.True(t, IsDuplicateKeyError(err), "got %v", err)

	existing, err := Query.FindOAuthClientByMetadataHash(ctx, database.RW(), metadataHash)
	require.NoError(t, err)
	require.Equal(t, first.ID, existing.ID)
	requireJSONStrings(t, existing.RedirectUris, []string{"https://gateway.test/oauth/callback"})

	byID, err := Query.FindOAuthClientByID(ctx, database.RW(), first.ID)
	require.NoError(t, err)
	require.Equal(t, "Unkey MCP", byID.Name)
	require.Equal(t, now, byID.CreatedAt)

	_, err = Query.FindOAuthClientByID(ctx, database.RW(), duplicate.ID)
	require.True(t, IsNotFound(err))
}

// TestCLISessionClientIDRoundTrip guarantees that a session records which
// OAuth client holds it, and that sessions inserted without one read back as
// held by the Unkey CLI.
func TestCLISessionClientIDRoundTrip(t *testing.T) {
	skipWithoutDatabase(t)
	ctx := context.Background()
	database := openCLILoginDB(t, containers.MySQL(t))
	now := time.Now().UnixMilli()
	workspaceID := uid.New(uid.WorkspacePrefix)
	userID := uid.New(uid.TestPrefix)
	clientID := uid.New(uid.OAuthClientPrefix, 16)

	permissions, err := json.Marshal([]string{"api.*.read_api"})
	require.NoError(t, err)
	sessionID := uid.New(uid.CLISessionPrefix)
	require.NoError(t, Query.InsertCLISession(ctx, database.RW(), InsertCLISessionParams{
		ID:               sessionID,
		WorkspaceID:      workspaceID,
		UserID:           userID,
		UserName:         sql.NullString{},
		Permissions:      permissions,
		DeviceName:       sql.NullString{String: "Unkey MCP", Valid: true},
		RefreshTokenHash: hash.Sha256(uid.New(uid.TestPrefix, 32)),
		RefreshFamily:    uid.New(uid.TestPrefix, 20),
		ClientID:         sql.NullString{String: clientID, Valid: true},
		CreatedAt:        now,
		LastUsedAt:       now,
		ExpiresAt:        now + time.Hour.Milliseconds(),
	}))
	cli := insertActiveCLISession(t, ctx, database, cliSessionFixture{
		WorkspaceID: workspaceID,
		UserID:      userID,
		CreatedAt:   now - time.Minute.Milliseconds(),
		ExpiresAt:   now + time.Hour.Milliseconds(),
	})

	held, err := Query.FindCLISessionByID(ctx, database.RW(), sessionID)
	require.NoError(t, err)
	require.Equal(t, sql.NullString{String: clientID, Valid: true}, held.ClientID)
	require.False(t, cli.ClientID.Valid)

	listed, err := Query.ListCLISessionsByUser(ctx, database.RO(), ListCLISessionsByUserParams{
		WorkspaceID: workspaceID,
		UserID:      userID,
		Now:         now,
	})
	require.NoError(t, err)
	require.Len(t, listed, 2)
	require.Equal(t, sessionID, listed[0].ID)
	require.Equal(t, sql.NullString{String: clientID, Valid: true}, listed[0].ClientID)
	require.False(t, listed[1].ClientID.Valid)
}
