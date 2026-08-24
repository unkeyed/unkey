package portal_test

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/internal/portal"
)

func TestColumnsForSetsExactlyOneColumn(t *testing.T) {
	t.Parallel()

	t.Run("app mapping sets only app_id", func(t *testing.T) {
		t.Parallel()

		appID, keyAuthID, err := portal.ColumnsFor(portal.Mapping{
			Type: portal.MappingTypeApp,
			ID:   "app_123",
		})
		require.NoError(t, err)
		require.Equal(t, sql.NullString{String: "app_123", Valid: true}, appID)
		require.False(t, keyAuthID.Valid, "keyspace column must stay null")
	})

	t.Run("keyspace mapping sets only key_auth_id", func(t *testing.T) {
		t.Parallel()

		appID, keyAuthID, err := portal.ColumnsFor(portal.Mapping{
			Type: portal.MappingTypeKeyspace,
			ID:   "ks_123",
		})
		require.NoError(t, err)
		require.Equal(t, sql.NullString{String: "ks_123", Valid: true}, keyAuthID)
		require.False(t, appID.Valid, "app column must stay null")
	})

	// The generated enum makes this unreachable through a well-formed request,
	// but the switch must stay total rather than silently defaulting to one arm.
	t.Run("rejects unknown type", func(t *testing.T) {
		t.Parallel()

		_, _, err := portal.ColumnsFor(portal.Mapping{
			Type: portal.MappingType("project"),
			ID:   "app_123",
		})
		require.Error(t, err)
	})
}

func TestMappingFrom(t *testing.T) {
	t.Parallel()

	ptr := func(v string) *string { return &v }

	t.Run("keyspace id alone", func(t *testing.T) {
		t.Parallel()

		mapping, err := portal.MappingFrom(ptr("ks_123"), nil)
		require.NoError(t, err)
		require.Equal(t, portal.Mapping{Type: portal.MappingTypeKeyspace, ID: "ks_123"}, mapping)
	})

	t.Run("app id alone", func(t *testing.T) {
		t.Parallel()

		mapping, err := portal.MappingFrom(nil, ptr("app_123"))
		require.NoError(t, err)
		require.Equal(t, portal.Mapping{Type: portal.MappingTypeApp, ID: "app_123"}, mapping)
	})

	// The flat wire pair can express both and neither, which the nested object
	// could not. These two cases are what the type buys back.
	t.Run("rejects both ids", func(t *testing.T) {
		t.Parallel()

		_, err := portal.MappingFrom(ptr("ks_123"), ptr("app_123"))
		require.Error(t, err)
	})

	t.Run("rejects neither id", func(t *testing.T) {
		t.Parallel()

		_, err := portal.MappingFrom(nil, nil)
		require.Error(t, err)
	})

	// A blank id would otherwise be written as a valid column holding "", which
	// claims the association without naming anything. Whitespace-only is the case
	// `oneOf` and minLength cannot catch, which is why this check is not left to
	// the schema.
	for name, id := range map[string]string{"empty": "", "whitespace": "   "} {
		t.Run("rejects "+name+" keyspace id", func(t *testing.T) {
			t.Parallel()

			_, err := portal.MappingFrom(ptr(id), nil)
			require.Error(t, err)
		})

		t.Run("rejects "+name+" app id", func(t *testing.T) {
			t.Parallel()

			_, err := portal.MappingFrom(nil, ptr(id))
			require.Error(t, err)
		})
	}

	// One blank and one real id is not ambiguous: the blank is not a value, so
	// the real one wins rather than the pair being refused.
	t.Run("blank id alongside a real one", func(t *testing.T) {
		t.Parallel()

		mapping, err := portal.MappingFrom(ptr("  "), ptr("app_123"))
		require.NoError(t, err)
		require.Equal(t, portal.Mapping{Type: portal.MappingTypeApp, ID: "app_123"}, mapping)
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
		require.Equal(t, portal.MappingTypeApp, mapping.Type)
		require.Equal(t, "app_123", mapping.ID)
	})

	t.Run("keyspace row", func(t *testing.T) {
		t.Parallel()

		mapping, err := portal.MappingOf(db.Portal{
			ID:        "pc_1",
			KeyAuthID: sql.NullString{String: "ks_123", Valid: true},
		})
		require.NoError(t, err)
		require.Equal(t, portal.MappingTypeKeyspace, mapping.Type)
		require.Equal(t, "ks_123", mapping.ID)
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
		require.NotNil(t, got.KeyspaceId)
		require.Equal(t, "ks_1", string(*got.KeyspaceId))
		require.Nil(t, got.AppId)
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
		require.NotNil(t, got.AppId)
		require.Equal(t, "app_1", string(*got.AppId))
		require.Nil(t, got.KeyspaceId)
	})

	t.Run("ambiguous row surfaces the error", func(t *testing.T) {
		t.Parallel()

		_, err := portal.ToResponse(db.Portal{ID: "pc_1", Slug: "acme"})
		require.Error(t, err)
	})
}

// DescribeMapping is the audit-log counterpart to MappingOf, and it must not
// share its refusal: a mutation on a row that already violates the invariant
// still has to be recorded, or a misconfigured portal could never be deleted.
func TestDescribeMappingNeverFails(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		row      db.Portal
		wantType string
		wantID   string
	}{
		"app mapped": {
			row:      db.Portal{AppID: sql.NullString{String: "app_1", Valid: true}},
			wantType: "app",
			wantID:   "app_1",
		},
		"keyspace mapped": {
			row:      db.Portal{KeyAuthID: sql.NullString{String: "ks_1", Valid: true}},
			wantType: "keyspace",
			wantID:   "ks_1",
		},
		// Both ids survive: an incident reviewer needs to know which resources the
		// row was claiming, which is exactly what a single "invalid" would hide.
		"both set is invalid and keeps both ids": {
			row: db.Portal{
				AppID:     sql.NullString{String: "app_1", Valid: true},
				KeyAuthID: sql.NullString{String: "ks_1", Valid: true},
			},
			wantType: "invalid",
			wantID:   "app_1,ks_1",
		},
		// Distinct from "invalid": no mapping at all is a different problem from
		// two conflicting ones, and collapsing them loses that.
		"neither set is none": {
			row:      db.Portal{},
			wantType: "none",
			wantID:   "",
		},
		"valid but empty column counts as absent": {
			row:      db.Portal{AppID: sql.NullString{String: "", Valid: true}},
			wantType: "none",
			wantID:   "",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			gotType, gotID := portal.DescribeMapping(tc.row)
			require.Equal(t, tc.wantType, gotType)
			require.Equal(t, tc.wantID, gotID)
		})
	}
}

// The comparison update uses to decide whether a mapping changed. Raw
// NullString equality would call a Valid-but-empty column different from NULL
// and revoke every live session for an identical mapping.
func TestSameAssociationTreatsEmptyAsAbsent(t *testing.T) {
	t.Parallel()

	absent := sql.NullString{String: "", Valid: false}
	emptyButValid := sql.NullString{String: "", Valid: true}
	set := sql.NullString{String: "app_1", Valid: true}
	other := sql.NullString{String: "app_2", Valid: true}

	require.True(t, portal.SameAssociation(absent, absent))
	require.True(t, portal.SameAssociation(set, set))
	require.True(t, portal.SameAssociation(absent, emptyButValid),
		"a Valid but empty column means the same as NULL: no association")
	require.False(t, portal.SameAssociation(set, other))
	require.False(t, portal.SameAssociation(set, absent))
}

// A row that already violates the one-mapping invariant must still be
// describable, or a mutation on it would roll back and the operator could never
// switch it off.
func TestToResponseTolerantNeverFails(t *testing.T) {
	t.Parallel()

	ambiguous := db.Portal{
		ID:        "pc_1",
		Slug:      "broken",
		Enabled:   true,
		AppID:     sql.NullString{String: "app_1", Valid: true},
		KeyAuthID: sql.NullString{String: "ks_1", Valid: true},
		CreatedAt: 1,
	}

	// ToResponse refuses it, which is why the tolerant variant exists.
	_, err := portal.ToResponse(ambiguous)
	require.Error(t, err)

	// The flat response carries at most one id, so it has no way to say "this row
	// claims two resources". It omits both rather than picking one or inventing a
	// type, and the audit entry keeps the real state through DescribeMapping.
	got := portal.ToResponseTolerant(ambiguous)
	require.Equal(t, "pc_1", got.Id)
	require.Equal(t, "broken", got.Slug)
	require.Nil(t, got.KeyspaceId, "an ambiguous row must not name a keyspace")
	require.Nil(t, got.AppId, "an ambiguous row must not name an app")

	unmapped := portal.ToResponseTolerant(db.Portal{ID: "pc_2", Slug: "none", CreatedAt: 1})
	require.Nil(t, unmapped.KeyspaceId)
	require.Nil(t, unmapped.AppId)
}

// The audit trail is where a corrupt row still has to be legible, since the
// response can no longer describe one.
func TestDescribeMappingStillNamesBrokenRows(t *testing.T) {
	t.Parallel()

	mappingType, mappingID := portal.DescribeMapping(db.Portal{
		ID:        "pc_1",
		AppID:     sql.NullString{String: "app_1", Valid: true},
		KeyAuthID: sql.NullString{String: "ks_1", Valid: true},
	})
	require.Equal(t, "invalid", mappingType)
	require.Equal(t, "app_1,ks_1", mappingID, "both ids survive so an incident reviewer sees the claim")

	mappingType, mappingID = portal.DescribeMapping(db.Portal{ID: "pc_2"})
	require.Equal(t, "none", mappingType)
	require.Empty(t, mappingID)
}
