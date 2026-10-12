package testutil

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	portalservice "github.com/unkeyed/unkey/internal/services/portal"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/portal"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
)

// SeedPortal creates a portal serving the given resource.
//
// The seeder takes the two association columns, and which one carries the id
// depends on the mapping kind, so every portal route test needs the same
// derivation. It lives here rather than being copied into each package.
//
// The project is derived from the mapping rather than passed in, so a seeded row
// satisfies the invariant the routes enforce, and the mapped resource must exist.
//
// A nil logoURL or primaryColor leaves that branding column absent, which is
// distinct from present-but-empty.
func (h *Harness) SeedPortal(
	t *testing.T,
	workspaceID, slug, displayName string,
	mapping portal.Mapping,
	logoURL, primaryColor *string,
) db.Portal {
	t.Helper()

	appID := sql.NullString{String: "", Valid: false}
	keyAuthID := sql.NullString{String: "", Valid: false}
	switch mapping.Type {
	case portal.MappingTypeApp:
		appID = sql.NullString{String: mapping.ID, Valid: true}
	case portal.MappingTypeKeyspace:
		keyAuthID = sql.NullString{String: mapping.ID, Valid: true}
	default:
		t.Fatalf("unsupported portal mapping type %q", mapping.Type)
	}

	projectID, err := portal.ResolveMappingProject(context.Background(), h.DB.RO(), workspaceID, mapping)
	require.NoError(t, err, "the mapped resource must exist for its project to be derived")

	return h.CreatePortal(seed.CreatePortalRequest{
		ID:           "",
		WorkspaceID:  workspaceID,
		ProjectID:    projectID,
		Slug:         slug,
		DisplayName:  displayName,
		AppID:        appID,
		KeyAuthID:    keyAuthID,
		Enabled:      true,
		LogoUrl:      nullableString(logoURL),
		PrimaryColor: nullableString(primaryColor),
	})
}

func nullableString(v *string) sql.NullString {
	if v == nil {
		return sql.NullString{String: "", Valid: false}
	}
	return sql.NullString{String: *v, Valid: true}
}

// RootKeyHeaders returns JSON request headers authenticated with rootKey.
func RootKeyHeaders(rootKey string) http.Header {
	return http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}
}

// SeedKeyspaceMapping seeds an api and returns a portal mapping to its keyspace,
// plus the project that owns it for URN grants.
func (h *Harness) SeedKeyspaceMapping(t *testing.T, workspaceID string) (portal.Mapping, string) {
	t.Helper()

	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID:   workspaceID,
		ProjectID:     "",
		IpWhitelist:   "",
		EncryptedKeys: false,
		Name:          nil,
		CreatedAt:     nil,
		DefaultPrefix: nil,
		DefaultBytes:  nil,
	})
	return portal.Mapping{Type: portal.MappingTypeKeyspace, ID: api.KeyAuthID.String}, api.ProjectID
}

// CreatePortalSessionInState inserts a pending or exchanged session expiring at
// expiresAt and returns its exchange code. Use it for the states
// [Harness.CreatePortalSessionForPortal] can't produce, such as pending or
// expired sessions.
func (h *Harness) CreatePortalSessionInState(t *testing.T, portalID, workspaceID, externalID string, exchanged bool, expiresAt time.Time) string {
	t.Helper()

	exchangeCode := string(uid.PortalExchangeCodePrefix) + "_" + uid.Secure()
	now := h.Clock.Now()
	ctx := context.Background()

	scopes, err := json.Marshal(portalservice.Grant{KeyspaceIDs: []string{}, Scopes: []string{"keys:read"}})
	require.NoError(t, err)

	require.NoError(t, db.Query.InsertPortalSession(ctx, h.DB.RW(), db.InsertPortalSessionParams{
		ID:                    uid.New(uid.PortalSessionPrefix),
		WorkspaceID:           workspaceID,
		PortalID:              portalID,
		ExternalID:            externalID,
		Scopes:                scopes,
		ExchangeCodeHash:      hash.Sha256(exchangeCode),
		ExchangeCodeExpiresAt: expiresAt.UnixMilli(),
		ReturnUrl:             sql.NullString{Valid: false, String: ""},
		CreatedAt:             now.UnixMilli(),
	}))

	if exchanged {
		accessToken := string(uid.PortalAccessTokenPrefix) + "_" + uid.Secure()
		res, err := db.Query.ExchangePortalSessionCode(ctx, h.DB.RW(), db.ExchangePortalSessionCodeParams{
			AccessTokenHash:      sql.NullString{String: hash.Sha256(accessToken), Valid: true},
			AccessTokenCreatedAt: sql.NullInt64{Int64: now.UnixMilli(), Valid: true},
			AccessTokenExpiresAt: sql.NullInt64{Int64: expiresAt.UnixMilli(), Valid: true},
			ExchangeCodeHash:     hash.Sha256(exchangeCode),
			// Redeem just before expiry so an already expired session can still be exchanged
			Now: expiresAt.UnixMilli() - 1,
		})
		require.NoError(t, err)
		redeemed, err := res.RowsAffected()
		require.NoError(t, err)
		require.Equal(t, int64(1), redeemed)
	}

	return exchangeCode
}

// CountLivePortalSessions counts a portal's unrevoked sessions, optionally for
// one end user when externalID is non-empty. Read from the primary so a test
// sees writes it just made.
func (h *Harness) CountLivePortalSessions(t *testing.T, portalID, externalID string) int {
	t.Helper()

	count, err := db.Query.CountLivePortalSessionsByPortal(context.Background(), h.DB.RW(), db.CountLivePortalSessionsByPortalParams{
		PortalID:   portalID,
		ExternalID: externalID,
	})
	require.NoError(t, err)
	return int(count)
}
