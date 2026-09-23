package migration_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/rbac/migration"
	"github.com/unkeyed/unkey/pkg/urn"
)

func TestTranslateScopedAPI(t *testing.T) {
	got, err := migration.Translate("api.api_source.decrypt_key", migration.Scope{
		WorkspaceID: "ws_customer",
		APIs: map[string]urn.V1{
			"api_source": {WorkspaceID: "ws_customer", Resource: "projects/proj_owner/keyspaces/ks_target"},
		},
	})
	require.NoError(t, err)
	require.Equal(t, "unkey:v1:ws_customer:projects/proj_owner/keyspaces/ks_target/keys/*#decrypt", got)
}

func TestTranslateNamespacePreservesIdentity(t *testing.T) {
	for _, tt := range []struct {
		resource string
		want     string
	}{
		{"projects/proj_limits/ratelimits/namespaces/ns_source", "unkey:v1:ws_customer:projects/proj_limits/ratelimits/namespaces/ns_source/overrides/*#write"},
		{"projects/proj_limits/ratelimits/namespaces/ns_other", ""},
		{"projects/proj_limits/ratelimits/namespaces/*", ""},
		{"projects/proj_limits/keyspaces/ks_other", ""},
	} {
		t.Run(tt.resource, func(t *testing.T) {
			got, err := migration.Translate("ratelimit.ns_source.set_override", migration.Scope{
				WorkspaceID: "ws_customer",
				Namespaces:  map[string]urn.V1{"ns_source": {WorkspaceID: "ws_customer", Resource: tt.resource}},
			})
			if tt.want == "" {
				require.ErrorIs(t, err, migration.ErrInvalidScope)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.want, got)
		})
	}
}

func TestTranslateRejectsUnsafeResourceResolution(t *testing.T) {
	for name, resource := range map[string]urn.V1{
		"foreign workspace":   {WorkspaceID: "ws_other", Resource: "projects/proj_owner/keyspaces/ks_target"},
		"project wildcard":    {WorkspaceID: "ws_customer", Resource: "projects/*/keyspaces/ks_target"},
		"keyspace wildcard":   {WorkspaceID: "ws_customer", Resource: "projects/proj_owner/keyspaces/*"},
		"wrong resource type": {WorkspaceID: "ws_customer", Resource: "projects/proj_owner/apps/app_target"},
		"subtree":             {WorkspaceID: "ws_customer", Resource: "projects/proj_owner/keyspaces/ks_target/**"},
		"missing mapping":     {},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := migration.Translate("api.api_source.decrypt_key", migration.Scope{
				WorkspaceID: "ws_customer",
				APIs:        map[string]urn.V1{"api_source": resource},
			})
			require.Error(t, err)
			require.Empty(t, got)
		})
	}
}

func TestTranslateRejectsInvalidInputs(t *testing.T) {
	for _, legacy := range []string{"", "api..decrypt_key", "api.api_*.decrypt_key", "api.**.decrypt_key", "api.id.decrypt_key.extra", "unkey:v1:ws_customer:**#*"} {
		t.Run(legacy, func(t *testing.T) {
			got, err := migration.Translate(legacy, migration.Scope{WorkspaceID: "ws_customer"})
			require.ErrorIs(t, err, migration.ErrUnsupported)
			require.Empty(t, got)
		})
	}
	for _, workspaceID := range []string{"", "*", "ws_customer:other"} {
		t.Run("workspace="+workspaceID, func(t *testing.T) {
			got, err := migration.Translate("*", migration.Scope{WorkspaceID: workspaceID})
			require.ErrorIs(t, err, migration.ErrInvalidScope)
			require.Empty(t, got)
		})
	}
	for _, legacy := range []string{"api.api_missing.decrypt_key", "ratelimit.ns_missing.limit"} {
		t.Run(legacy, func(t *testing.T) {
			got, err := migration.Translate(legacy, migration.Scope{WorkspaceID: "ws_customer"})
			require.ErrorIs(t, err, migration.ErrInvalidScope)
			require.Empty(t, got)
		})
	}
}
