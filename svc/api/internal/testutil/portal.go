package testutil

import (
	"database/sql"
	"testing"

	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/internal/portal"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
)

// SeedPortal creates a portal serving the given resource.
//
// The seeder takes the two association columns, and which one carries the id
// depends on the mapping kind, so every portal route test needs the same
// derivation. It lives here rather than being copied into each package.
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

	return h.CreatePortal(seed.CreatePortalRequest{
		ID:           "",
		WorkspaceID:  workspaceID,
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
