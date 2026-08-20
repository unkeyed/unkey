package portal_test

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/internal/portal"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestColumnsForSetsExactlyOneColumn(t *testing.T) {
	t.Parallel()

	t.Run("app mapping sets only app_id", func(t *testing.T) {
		t.Parallel()

		appID, keyAuthID, err := portal.ColumnsFor(openapi.PortalMapping{
			Id:   "app_123",
			Type: openapi.PortalMappingTypeApp,
		})
		require.NoError(t, err)
		require.Equal(t, sql.NullString{String: "app_123", Valid: true}, appID)
		require.False(t, keyAuthID.Valid, "keyspace column must stay null")
	})

	t.Run("keyspace mapping sets only key_auth_id", func(t *testing.T) {
		t.Parallel()

		appID, keyAuthID, err := portal.ColumnsFor(openapi.PortalMapping{
			Id:   "ks_123",
			Type: openapi.PortalMappingTypeKeyspace,
		})
		require.NoError(t, err)
		require.Equal(t, sql.NullString{String: "ks_123", Valid: true}, keyAuthID)
		require.False(t, appID.Valid, "app column must stay null")
	})

	// An empty or whitespace id would otherwise be written as a valid column
	// holding "", which claims the association without naming anything.
	for name, id := range map[string]string{"empty": "", "whitespace": "   "} {
		t.Run("rejects "+name+" id", func(t *testing.T) {
			t.Parallel()

			_, _, err := portal.ColumnsFor(openapi.PortalMapping{
				Id:   id,
				Type: openapi.PortalMappingTypeApp,
			})
			require.Error(t, err)
		})
	}

	// The generated enum makes this unreachable through a well-formed request,
	// but the switch must stay total rather than silently defaulting to one arm.
	t.Run("rejects unknown type", func(t *testing.T) {
		t.Parallel()

		_, _, err := portal.ColumnsFor(openapi.PortalMapping{
			Id:   "app_123",
			Type: openapi.PortalMappingType("project"),
		})
		require.Error(t, err)
	})
}

func TestMappingOfRejectsAmbiguousRows(t *testing.T) {
	t.Parallel()

	t.Run("app row", func(t *testing.T) {
		t.Parallel()

		mapping, err := portal.MappingOf(db.Portal{
			ID:    "pc_1",
			AppID: sql.NullString{String: "app_123", Valid: true},
		})
		require.NoError(t, err)
		require.Equal(t, openapi.PortalMappingTypeApp, mapping.Type)
		require.Equal(t, "app_123", mapping.Id)
	})

	t.Run("keyspace row", func(t *testing.T) {
		t.Parallel()

		mapping, err := portal.MappingOf(db.Portal{
			ID:        "pc_1",
			KeyAuthID: sql.NullString{String: "ks_123", Valid: true},
		})
		require.NoError(t, err)
		require.Equal(t, openapi.PortalMappingTypeKeyspace, mapping.Type)
		require.Equal(t, "ks_123", mapping.Id)
	})

	// Rows written before these routes existed were never checked against the
	// invariant, so both of these are reachable in real data.
	t.Run("both columns set is refused", func(t *testing.T) {
		t.Parallel()

		_, err := portal.MappingOf(db.Portal{
			ID:        "pc_1",
			AppID:     sql.NullString{String: "app_123", Valid: true},
			KeyAuthID: sql.NullString{String: "ks_123", Valid: true},
		})
		require.Error(t, err)
	})

	t.Run("neither column set is refused", func(t *testing.T) {
		t.Parallel()

		_, err := portal.MappingOf(db.Portal{ID: "pc_1"})
		require.Error(t, err)
	})

	// A column that is Valid but empty is the same ambiguity wearing a different
	// hat: it claims the association without naming a resource.
	t.Run("empty-string column counts as absent", func(t *testing.T) {
		t.Parallel()

		_, err := portal.MappingOf(db.Portal{
			ID:    "pc_1",
			AppID: sql.NullString{String: "", Valid: true},
		})
		require.Error(t, err)
	})
}

func TestValidateLogoURL(t *testing.T) {
	t.Parallel()

	valid := []string{
		"https://cdn.example.com/logo.svg",
		"https://example.com",
		"https://" + strings.Repeat("a", 480) + ".com",
	}
	for _, raw := range valid {
		require.NoError(t, portal.ValidateLogoURL(raw), "expected %q to be accepted", raw)
	}

	invalid := map[string]string{
		"http scheme":       "http://cdn.example.com/logo.svg",
		"no scheme":         "cdn.example.com/logo.svg",
		"not a url":         "this is not a url",
		"scheme only":       "https://",
		"javascript scheme": "javascript:alert(1)",
		"data scheme":       "data:image/svg+xml;base64,AAAA",
		"empty":             "",
		// One over the column width. Without this the driver would either
		// truncate the value, silently changing which host is contacted, or fail
		// and surface as a 500 rather than a validation error.
		"too long": "https://example.com/" + strings.Repeat("a", portal.LogoURLMaxLength),
	}
	for name, raw := range invalid {
		require.Error(t, portal.ValidateLogoURL(raw), "expected %s to be rejected", name)
	}

	require.Len(t, "https://example.com/"+strings.Repeat("a", portal.LogoURLMaxLength-len("https://example.com/")),
		portal.LogoURLMaxLength, "boundary fixture is exactly at the limit")
	require.NoError(t, portal.ValidateLogoURL(
		"https://example.com/"+strings.Repeat("a", portal.LogoURLMaxLength-len("https://example.com/"))),
		"a url exactly at the limit is accepted")
}

func TestValidatePrimaryColor(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{"#6366f1", "#FFFFFF", "#000000", "#AbCdEf"} {
		require.NoError(t, portal.ValidatePrimaryColor(raw), "expected %q to be accepted", raw)
	}

	invalid := map[string]string{
		"three-digit shorthand": "#fff",
		"missing hash":          "6366f1",
		"named colour":          "rebeccapurple",
		"eight digits":          "#6366f1ff",
		"non-hex character":     "#6366fg",
		"empty":                 "",
		"css function":          "rgb(99,102,241)",
	}
	for name, raw := range invalid {
		require.Error(t, portal.ValidatePrimaryColor(raw), "expected %s to be rejected", name)
	}
}

func TestToResponseOmitsAbsentBranding(t *testing.T) {
	t.Parallel()

	t.Run("no branding columns omits the object", func(t *testing.T) {
		t.Parallel()

		got, err := portal.ToResponse(db.Portal{
			ID:        "pc_1",
			Slug:      "acme",
			Enabled:   true,
			KeyAuthID: sql.NullString{String: "ks_1", Valid: true},
			CreatedAt: 1719849600000,
		})
		require.NoError(t, err)
		require.Nil(t, got.Branding, "branding must be absent rather than two empty strings")
		require.Equal(t, "acme", got.Slug)
		require.Equal(t, openapi.PortalMappingTypeKeyspace, got.Mapping.Type)
		require.Zero(t, got.UpdatedAt, "a never-updated portal reports no update time")
	})

	t.Run("one branding column present includes the object", func(t *testing.T) {
		t.Parallel()

		got, err := portal.ToResponse(db.Portal{
			ID:        "pc_1",
			Slug:      "acme",
			Enabled:   true,
			AppID:     sql.NullString{String: "app_1", Valid: true},
			LogoUrl:   sql.NullString{String: "https://cdn.example.com/l.svg", Valid: true},
			CreatedAt: 1719849600000,
			UpdatedAt: sql.NullInt64{Int64: 1719936000000, Valid: true},
		})
		require.NoError(t, err)
		require.NotNil(t, got.Branding)
		require.Equal(t, "https://cdn.example.com/l.svg", got.Branding.LogoUrl)
		require.Empty(t, got.Branding.PrimaryColor, "an unset colour stays empty rather than defaulting")
		require.Equal(t, int64(1719936000000), got.UpdatedAt)
	})

	// The response shape carries no display name at all: the name an operator
	// sees is resolved from the mapped app or keyspace, so a rename there cannot
	// leave a stale copy on the portal.
	t.Run("response carries the mapping ids, not a name", func(t *testing.T) {
		t.Parallel()

		got, err := portal.ToResponse(db.Portal{
			ID:        "pc_1",
			Slug:      "acme",
			AppID:     sql.NullString{String: "app_1", Valid: true},
			CreatedAt: 1,
		})
		require.NoError(t, err)
		require.Equal(t, "app_1", got.Mapping.Id)
	})

	t.Run("ambiguous row surfaces the error", func(t *testing.T) {
		t.Parallel()

		_, err := portal.ToResponse(db.Portal{ID: "pc_1", Slug: "acme"})
		require.Error(t, err)
	})
}
