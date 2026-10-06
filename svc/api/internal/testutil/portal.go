package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

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

	require.NoError(t, db.Query.InsertPortalSession(ctx, h.DB.RW(), db.InsertPortalSessionParams{
		ID:                    uid.New(uid.PortalSessionPrefix),
		WorkspaceID:           workspaceID,
		PortalID:              portalID,
		ExternalID:            externalID,
		Scopes:                []byte(`{"keyspaceIds":[],"scopes":["keys:read"]}`),
		ExchangeCodeHash:      hash.Sha256(exchangeCode),
		ExchangeCodeExpiresAt: expiresAt.UnixMilli(),
		ReturnUrl:             sql.NullString{Valid: false, String: ""},
		CreatedAt:             now.UnixMilli(),
	}))

	if exchanged {
		accessToken := string(uid.PortalAccessTokenPrefix) + "_" + uid.Secure()
		_, err := h.DB.RW().ExecContext(ctx,
			"UPDATE portal_sessions SET access_token_hash = ?, access_token_created_at = ?, access_token_expires_at = ? WHERE exchange_code_hash = ?",
			hash.Sha256(accessToken), now.UnixMilli(), expiresAt.UnixMilli(), hash.Sha256(exchangeCode),
		)
		require.NoError(t, err)
	}

	return exchangeCode
}

// CountLivePortalSessions counts a portal's unrevoked sessions, optionally for
// one end user when externalID is non-empty. Read from the primary so a test
// sees writes it just made.
func (h *Harness) CountLivePortalSessions(t *testing.T, portalID, externalID string) int {
	t.Helper()

	query := "SELECT COUNT(*) FROM portal_sessions WHERE portal_id = ? AND revoked_at IS NULL"
	args := []any{portalID}
	if externalID != "" {
		query += " AND external_id = ?"
		args = append(args, externalID)
	}

	var count int
	require.NoError(t, h.DB.RW().QueryRowContext(context.Background(), query, args...).Scan(&count))
	return count
}

// SeedPortalDomain inserts a domain on portalID in the given status and returns
// it as the API reads it. ctrl owns the insert query, so the row is written
// directly here.
func (h *Harness) SeedPortalDomain(
	t *testing.T,
	workspaceID, portalID, domain string,
	status db.PortalDomainsVerificationStatus,
) db.FindPortalDomainByIdRow {
	t.Helper()

	ctx := context.Background()
	id := uid.New(uid.PortalDomainPrefix)
	_, err := h.DB.RW().ExecContext(ctx,
		`INSERT INTO portal_domains (id, workspace_id, portal_id, domain, verification_status, verification_token, target_cname, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, workspaceID, portalID, domain, string(status), uid.Secure(24), uid.DNS1035(16)+".portal.unkey.local", h.Clock.Now().UnixMilli(),
	)
	require.NoError(t, err)

	row, err := db.Query.FindPortalDomainById(ctx, h.DB.RW(), db.FindPortalDomainByIdParams{
		ID:          id,
		WorkspaceID: workspaceID,
		PortalID:    portalID,
	})
	require.NoError(t, err)
	return row
}
