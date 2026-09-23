package migration_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/rbac/migration"
	"github.com/unkeyed/unkey/pkg/urn"
)

// These expectations cover the catalog independently of which grants happen
// to exist in the production export. IDs differ to expose incorrect parents.
var catalogPermissions = map[string]string{
	"api.api_source.read_api":                             "projects/proj_keys/keyspaces/ks_target#read",
	"api.api_source.delete_api":                           "projects/proj_keys/keyspaces/ks_target#delete",
	"api.api_source.read_key":                             "projects/proj_keys/keyspaces/ks_target/keys/*#read",
	"api.api_source.read_analytics":                       "projects/proj_keys/keyspaces/ks_target/logs#read",
	"api.api_source.decrypt_key":                          "projects/proj_keys/keyspaces/ks_target/keys/*#decrypt",
	"api.api_source.delete_key":                           "projects/proj_keys/keyspaces/ks_target/keys/*#delete",
	"api.api_source.verify_key":                           "projects/proj_keys/keyspaces/ks_target/keys/*#verify",
	"ratelimit.ns_source.limit":                           "projects/proj_limits/ratelimits/namespaces/ns_source#limit",
	"ratelimit.ns_source.read_override":                   "projects/proj_limits/ratelimits/namespaces/ns_source/overrides/*#read",
	"ratelimit.ns_source.set_override":                    "projects/proj_limits/ratelimits/namespaces/ns_source/overrides/*#write",
	"ratelimit.ns_source.delete_override":                 "projects/proj_limits/ratelimits/namespaces/ns_source/overrides/*#delete",
	"api.api_source.create_api":                           "projects/proj_keys/keyspaces/ks_target#write",
	"api.api_source.update_api":                           "projects/proj_keys/keyspaces/ks_target#write",
	"api.api_source.create_key":                           "projects/proj_keys/keyspaces/ks_target/keys/*#write",
	"api.api_source.update_key":                           "projects/proj_keys/keyspaces/ks_target/keys/*#write",
	"api.api_source.encrypt_key":                          "projects/proj_keys/keyspaces/ks_target/keys/*#write",
	"ratelimit.ns_source.create_namespace":                "projects/proj_limits/ratelimits/namespaces/ns_source#write",
	"ratelimit.ns_source.update_namespace":                "projects/proj_limits/ratelimits/namespaces/ns_source#write",
	"ratelimit.ns_source.list_overrides":                  "projects/proj_limits/ratelimits/namespaces/ns_source/overrides/*#read",
	"ratelimit.ns_source.read_namespace":                  "projects/proj_limits/ratelimits/namespaces/ns_source#read",
	"ratelimit.ns_source.delete_namespace":                "projects/proj_limits/ratelimits/namespaces/ns_source#delete",
	"ratelimit.ns_source.read_analytics":                  "projects/proj_limits/ratelimits/namespaces/ns_source/logs#read",
	"project.proj_source.read_project":                    "projects/proj_source#read",
	"project.proj_source.delete_project":                  "projects/proj_source#delete",
	"project.proj_source.read_deployment":                 "projects/proj_source/apps/*/environments/*/deployments/*#read",
	"project.proj_source.read_runtime_logs":               "projects/proj_source/apps/*/environments/*/deployments/*/logs#read",
	"project.proj_source.read_gateway_requests":           "projects/proj_source/apps/*/environments/*/gateway/logs#read",
	"app.app_source.read_app":                             "projects/proj_apps/apps/app_source#read",
	"app.app_source.delete_app":                           "projects/proj_apps/apps/app_source#delete",
	"environment.env_source.read_environment":             "projects/proj_env/apps/app_owner/environments/env_source#read",
	"environment.env_source.read_deployment":              "projects/proj_env/apps/app_owner/environments/env_source/deployments/*#read",
	"environment.env_source.read_environment_variables":   "projects/proj_env/apps/app_owner/environments/env_source/variables/*#read",
	"environment.env_source.remove_environment_variables": "projects/proj_env/apps/app_owner/environments/env_source/variables/*#delete",
	"environment.env_source.read_policies":                "projects/proj_env/apps/app_owner/environments/env_source/gateway/policies/*#read",
	"environment.env_source.read_domain":                  "projects/proj_env/apps/app_owner/environments/env_source/domains/*#read",
	"environment.env_source.delete_domain":                "projects/proj_env/apps/app_owner/environments/env_source/domains/*#delete",
	"identity.id_source.read_identity":                    "projects/proj_identity/identities/id_source#read",
	"identity.id_source.delete_identity":                  "projects/proj_identity/identities/id_source#delete",
	"rbac.*.read_role":                                    "projects/*/rbac/roles/*#read",
	"rbac.*.delete_role":                                  "projects/*/rbac/roles/*#delete",
	"rbac.*.read_permission":                              "projects/*/rbac/permissions/*#read",
	"rbac.*.delete_permission":                            "projects/*/rbac/permissions/*#delete",
	"project.proj_source.create_project":                  "projects/proj_source#write",
	"project.proj_source.update_project":                  "projects/proj_source#write",
	"project.proj_source.create_app":                      "projects/proj_source/apps/*#write",
	"project.proj_source.create_deployment":               "projects/proj_source/apps/*/environments/*/deployments/*#write",
	"project.proj_source.generate_upload_url":             "projects/proj_source/apps/*/environments/*/deployments/*#write",
	"app.app_source.update_app":                           "projects/proj_apps/apps/app_source#write",
	"app.app_source.connect_repository":                   "projects/proj_apps/apps/app_source#write",
	"environment.env_source.update_environment":           "projects/proj_env/apps/app_owner/environments/env_source#write",
	"environment.env_source.promote_deployment":           "projects/proj_env/apps/app_owner/environments/env_source#write",
	"environment.env_source.rollback_deployment":          "projects/proj_env/apps/app_owner/environments/env_source#write",
	"environment.env_source.create_deployment":            "projects/proj_env/apps/app_owner/environments/env_source/deployments/*#write",
	"environment.env_source.start_deployment":             "projects/proj_env/apps/app_owner/environments/env_source/deployments/*#write",
	"environment.env_source.stop_deployment":              "projects/proj_env/apps/app_owner/environments/env_source/deployments/*#write",
	"environment.env_source.set_environment_variables":    "projects/proj_env/apps/app_owner/environments/env_source/variables/*#write",
	"environment.env_source.set_policies":                 "projects/proj_env/apps/app_owner/environments/env_source/gateway/policies/*#write",
	"environment.env_source.update_policy":                "projects/proj_env/apps/app_owner/environments/env_source/gateway/policies/*#write",
	"environment.env_source.create_domain":                "projects/proj_env/apps/app_owner/environments/env_source/domains/*#write",
	"environment.env_source.verify_domain":                "projects/proj_env/apps/app_owner/environments/env_source/domains/*#write",
	"identity.id_source.create_identity":                  "projects/proj_identity/identities/id_source#write",
	"identity.id_source.update_identity":                  "projects/proj_identity/identities/id_source#write",
	"rbac.*.create_permission":                            "projects/*/rbac/permissions/*#write",
	"rbac.*.update_permission":                            "projects/*/rbac/permissions/*#write",
	"rbac.*.create_role":                                  "projects/*/rbac/roles/*#write",
	"rbac.*.update_role":                                  "projects/*/rbac/roles/*#write",
	"rbac.*.add_permission_to_role":                       "projects/*/rbac/roles/*#write",
	"rbac.*.remove_permission_from_role":                  "projects/*/rbac/roles/*#write",
	"rbac.*.add_permission_to_key":                        "projects/*/keyspaces/*/keys/*#write",
	"rbac.*.remove_permission_from_key":                   "projects/*/keyspaces/*/keys/*#write",
	"rbac.*.add_role_to_key":                              "projects/*/keyspaces/*/keys/*#write",
	"rbac.*.remove_role_from_key":                         "projects/*/keyspaces/*/keys/*#write",
	"workspace.*.install_github":                          "github/apps/*#write",
	"workspace.*.create_root_key":                         "rootKeys/*#write",
}

func catalogScope() migration.Scope {
	return migration.Scope{
		WorkspaceID:  "ws_customer",
		APIs:         map[string]urn.V1{"api_source": {WorkspaceID: "ws_customer", Resource: "projects/proj_keys/keyspaces/ks_target"}},
		Namespaces:   map[string]urn.V1{"ns_source": {WorkspaceID: "ws_customer", Resource: "projects/proj_limits/ratelimits/namespaces/ns_source"}},
		Apps:         map[string]urn.V1{"app_source": {WorkspaceID: "ws_customer", Resource: "projects/proj_apps/apps/app_source"}},
		Environments: map[string]urn.V1{"env_source": {WorkspaceID: "ws_customer", Resource: "projects/proj_env/apps/app_owner/environments/env_source"}},
		Identities:   map[string]urn.V1{"id_source": {WorkspaceID: "ws_customer", Resource: "projects/proj_identity/identities/id_source"}},
	}
}

func TestTranslateCatalog(t *testing.T) {
	for legacy, path := range catalogPermissions {
		t.Run(legacy, func(t *testing.T) {
			got, err := migration.Translate(legacy, catalogScope())
			require.NoError(t, err)
			require.Equal(t, "unkey:v1:ws_customer:"+path, got)
		})
	}
}

func TestTranslateCatalogWildcards(t *testing.T) {
	for legacy, path := range catalogPermissions {
		parts := strings.Split(legacy, ".")
		wildcard := parts[0] + ".*." + parts[2]
		want := strings.NewReplacer(
			"proj_keys", "*", "ks_target", "*",
			"proj_limits", "*", "ns_source", "*", "proj_source", "*",
			"proj_apps", "*", "app_source", "*", "proj_env", "*",
			"app_owner", "*", "env_source", "*", "proj_identity", "*", "id_source", "*",
		).Replace(path)
		t.Run(wildcard, func(t *testing.T) {
			got, err := migration.Translate(wildcard, migration.Scope{WorkspaceID: "ws_customer"})
			require.NoError(t, err)
			require.Equal(t, "unkey:v1:ws_customer:"+want, got)
		})
	}
}

func TestTranslateRejectsPartialMatches(t *testing.T) {
	for legacy := range catalogPermissions {
		for _, malformed := range []string{"prefix." + legacy, legacy + ".extra", legacy + "\n"} {
			t.Run(malformed, func(t *testing.T) {
				got, err := migration.Translate(malformed, catalogScope())
				require.ErrorIs(t, err, migration.ErrUnsupported)
				require.Empty(t, got)
			})
		}
	}
}

func TestTranslateRejectsInvalidResolvedResources(t *testing.T) {
	for _, family := range []struct {
		legacy    string
		id        string
		path      string
		resources func(migration.Scope) map[string]urn.V1
	}{
		{"app.app_source.update_app", "app_source", "projects/proj_apps/apps/app_source", func(s migration.Scope) map[string]urn.V1 { return s.Apps }},
		{"environment.env_source.promote_deployment", "env_source", "projects/proj_env/apps/app_owner/environments/env_source", func(s migration.Scope) map[string]urn.V1 { return s.Environments }},
		{"identity.id_source.delete_identity", "id_source", "projects/proj_identity/identities/id_source", func(s migration.Scope) map[string]urn.V1 { return s.Identities }},
	} {
		for name, resource := range map[string]urn.V1{
			"missing":           {},
			"foreign workspace": {WorkspaceID: "ws_other", Resource: family.path},
			"different id":      {WorkspaceID: "ws_customer", Resource: strings.ReplaceAll(family.path, family.id, "other")},
			"wildcard id":       {WorkspaceID: "ws_customer", Resource: strings.ReplaceAll(family.path, family.id, "*")},
			"wildcard parent":   {WorkspaceID: "ws_customer", Resource: strings.Replace(family.path, strings.Split(family.path, "/")[1], "*", 1)},
			"wrong type":        {WorkspaceID: "ws_customer", Resource: "projects/proj_wrong/keyspaces/" + family.id},
			"subtree":           {WorkspaceID: "ws_customer", Resource: family.path + "/**"},
		} {
			t.Run(family.legacy+"/"+name, func(t *testing.T) {
				scope := catalogScope()
				family.resources(scope)[family.id] = resource
				got, err := migration.Translate(family.legacy, scope)
				require.ErrorIs(t, err, migration.ErrInvalidScope)
				require.Empty(t, got)
			})
		}
	}
}

func TestTranslateRejectsResourcesWithoutCanonicalMapping(t *testing.T) {
	for _, legacy := range []string{
		"portal.*.create_portal", "portal.pc_source.read_portal", "portal.*.update_portal",
		"portal.pc_source.delete_portal", "portal.*.create_portal_session",
		"rbac.rbac_source.read_role", "workspace.ws_other.install_github", "api.*.unknown",
	} {
		t.Run(legacy, func(t *testing.T) {
			got, err := migration.Translate(legacy, catalogScope())
			require.ErrorIs(t, err, migration.ErrUnsupported)
			require.Empty(t, got)
		})
	}
}
