package gdprerasure_test

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/harness"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// seededIdentity is one identity of an end user and the key linked to it.
type seededIdentity struct {
	identityID string
	keyID      string
}

// TestEndUserErasureRemovesEveryIdentityAndSparesNeighbours guarantees that a
// GDPR end user erasure removes every identity with the external id, the
// soft-deleted one included, together with their keys, ratelimits, portal
// sessions and analytics rows, while another end user in the same workspace
// keeps everything. A second request deletes nothing.
func TestEndUserErasureRemovesEveryIdentityAndSparesNeighbours(t *testing.T) {
	h := harness.New(t)
	ctx := h.Ctx

	ws := h.Seed.CreateWorkspace(ctx)
	keySpaceID := h.Seed.CreateAPI(ctx, seed.CreateApiRequest{WorkspaceID: ws.ID}).KeyAuthID.String

	externalID := uid.New(uid.TestPrefix)
	deleted := seedEndUserIdentity(t, h, ws.ID, keySpaceID, externalID)
	require.NoError(t, h.DB.SoftDeleteIdentity(ctx, db.SoftDeleteIdentityParams{
		UpdatedAt: sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
		ID:        deleted.identityID,
	}))
	live := seedEndUserIdentity(t, h, ws.ID, keySpaceID, externalID)

	neighbourExternalID := uid.New(uid.TestPrefix)
	neighbour := seedEndUserIdentity(t, h, ws.ID, keySpaceID, neighbourExternalID)

	subjectIdentityIDs := []string{deleted.identityID, live.identityID}
	for _, identity := range []seededIdentity{deleted, live} {
		counts := countEndUserRows(t, h, ws.ID, externalID, identity)
		require.Equal(t, int64(2), counts.Identities)
		require.Positive(t, counts.KeyRows)
		require.Positive(t, counts.EncryptedKeys)
		require.Positive(t, counts.IdentityRatelimits)
		require.Positive(t, counts.KeyRatelimits)
		require.Positive(t, counts.PortalSessions)
	}
	require.NotEmpty(t, clickhouseCounts(t, h, "identity_id", "workspace_id = ? AND has(?, identity_id)", ws.ID, subjectIdentityIDs))
	require.NotEmpty(t, clickhouseCounts(t, h, "external_id", "workspace_id = ? AND external_id = ?", ws.ID, externalID))
	neighbourBefore := countEndUserRows(t, h, ws.ID, neighbourExternalID, neighbour)
	neighbourByIdentityBefore := clickhouseCounts(t, h, "identity_id", "workspace_id = ? AND identity_id = ?", ws.ID, neighbour.identityID)
	neighbourByExternalIDBefore := clickhouseCounts(t, h, "external_id", "workspace_id = ? AND external_id = ?", ws.ID, neighbourExternalID)

	_, err := erase(t, h, endUserRequest(ws.ID, externalID))
	require.NoError(t, err)

	for _, identity := range []seededIdentity{deleted, live} {
		require.Equal(t, db.CountIdentityRowsByTableRow{}, countEndUserRows(t, h, ws.ID, externalID, identity))
	}
	require.Empty(t, clickhouseCounts(t, h, "identity_id", "workspace_id = ? AND has(?, identity_id)", ws.ID, subjectIdentityIDs))
	require.Empty(t, clickhouseCounts(t, h, "external_id", "workspace_id = ? AND external_id = ?", ws.ID, externalID))

	require.Equal(t, neighbourBefore, countEndUserRows(t, h, ws.ID, neighbourExternalID, neighbour))
	require.Equal(t, neighbourByIdentityBefore, clickhouseCounts(t, h, "identity_id", "workspace_id = ? AND identity_id = ?", ws.ID, neighbour.identityID))
	require.Equal(t, neighbourByExternalIDBefore, clickhouseCounts(t, h, "external_id", "workspace_id = ? AND external_id = ?", ws.ID, neighbourExternalID))

	again, err := erase(t, h, endUserRequest(ws.ID, externalID))
	require.NoError(t, err)
	requireNothingLeft(t, again)
}

func endUserRequest(workspaceID, externalID string) *hydrav1.GdprErasureRequest {
	return &hydrav1.GdprErasureRequest{Subject: &hydrav1.GdprErasureRequest_EndUser{
		EndUser: &hydrav1.GdprErasureEndUser{WorkspaceId: workspaceID, ExternalId: externalID},
	}}
}

// seedEndUserIdentity creates an identity for externalID with a ratelimit, a
// key with permissions, roles, a ratelimit and a recoverable copy, a portal
// session, and verifications. One verification carries only the identity id
// and one only the external id, so each erasure match is exercised alone.
func seedEndUserIdentity(t *testing.T, h *harness.Harness, workspaceID, keySpaceID, externalID string) seededIdentity {
	t.Helper()
	ctx := h.Ctx
	now := time.Now()

	identityID := h.Seed.CreateIdentity(ctx, seed.CreateIdentityRequest{
		WorkspaceID: workspaceID,
		ExternalID:  externalID,
		Ratelimits:  []seed.CreateRatelimitRequest{{Name: "identity", WorkspaceID: workspaceID, Limit: 10, Duration: 1000}},
	})
	key := h.Seed.CreateKey(ctx, seed.CreateKeyRequest{
		WorkspaceID: workspaceID,
		KeySpaceID:  keySpaceID,
		IdentityID:  &identityID,
		Recoverable: true,
		Permissions: []seed.CreatePermissionRequest{{Name: uid.New(uid.TestPrefix), Slug: uid.New(uid.TestPrefix), WorkspaceID: workspaceID}},
		Roles:       []seed.CreateRoleRequest{{Name: uid.New(uid.TestPrefix), WorkspaceID: workspaceID}},
		Ratelimits:  []seed.CreateRatelimitRequest{{Name: "key", WorkspaceID: workspaceID, Limit: 10, Duration: 1000}},
	})
	require.NoError(t, h.DB.InsertPortalSession(ctx, db.InsertPortalSessionParams{
		ID:                    uid.New(uid.PortalSessionPrefix),
		WorkspaceID:           workspaceID,
		PortalID:              uid.New(uid.PortalPrefix),
		ExternalID:            externalID,
		Scopes:                json.RawMessage("[]"),
		ExchangeCodeHash:      uid.New(uid.TestPrefix),
		ExchangeCodeExpiresAt: now.Add(time.Minute).UnixMilli(),
		CreatedAt:             now.UnixMilli(),
	}))

	verifications := []schema.KeyVerification{
		{IdentityID: identityID, ExternalID: externalID},
		{IdentityID: identityID, ExternalID: ""},
		{IdentityID: "", ExternalID: externalID},
	}
	batch, err := h.ClickHouseConn.PrepareBatch(ctx, clickhouse.InsertQuery[schema.KeyVerification]())
	require.NoError(t, err)
	for i := range verifications {
		verifications[i].RequestID = uid.New(uid.RequestPrefix)
		verifications[i].Time = now.UnixMilli()
		verifications[i].WorkspaceID = workspaceID
		verifications[i].KeySpaceID = keySpaceID
		verifications[i].KeyID = key.KeyID
		verifications[i].Region = "us-east-1"
		verifications[i].Source = schema.SourceAPI
		verifications[i].Outcome = "VALID"
		verifications[i].Tags = []string{}
		require.NoError(t, batch.AppendStruct(&verifications[i]))
	}
	require.NoError(t, batch.Send())

	return seededIdentity{identityID: identityID, keyID: key.KeyID}
}

func countEndUserRows(t *testing.T, h *harness.Harness, workspaceID, externalID string, identity seededIdentity) db.CountIdentityRowsByTableRow {
	t.Helper()
	counts, err := h.DB.CountIdentityRowsByTable(h.Ctx, db.CountIdentityRowsByTableParams{
		WorkspaceID:    workspaceID,
		ExternalID:     externalID,
		IdentityID:     sql.NullString{String: identity.identityID, Valid: true},
		KeyID:          identity.keyID,
		RatelimitKeyID: sql.NullString{String: identity.keyID, Valid: true},
	})
	require.NoError(t, err)
	return counts
}
