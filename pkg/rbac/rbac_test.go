package rbac

import (
	"testing"
)

func TestRBAC_EvaluatePermissions(t *testing.T) {
	tests := []struct {
		name        string
		query       PermissionQuery
		permissions []string
		wantValid   bool
	}{
		{
			name:        "Simple permission check (Pass)",
			query:       T(Tuple{ResourceType: Api, ResourceID: "api1", Action: ReadAPI}),
			permissions: []string{"api.api1.read_api", "api.api1.update_api", "rbac.role1.read_role"},
			wantValid:   true,
		},
		{
			name:        "Simple permission check (Fail)",
			query:       T(Tuple{ResourceType: Api, ResourceID: "api2", Action: ReadAPI}),
			permissions: []string{"api.api1.read_api", "api.api1.update_api", "rbac.role1.read_role"},
			wantValid:   false,
		},
		{
			name: "AND of two permissions (Pass)",
			query: And(
				T(Tuple{ResourceType: Api, ResourceID: "api1", Action: ReadAPI}),
				T(Tuple{ResourceType: Api, ResourceID: "api1", Action: UpdateAPI}),
			),
			permissions: []string{"api.api1.read_api", "api.api1.update_api", "rbac.role1.read_role"},
			wantValid:   true,
		},
		{
			name: "OR of two permissions (Pass)",
			query: Or(
				T(Tuple{ResourceType: Api, ResourceID: "api1", Action: ReadAPI}),
				T(Tuple{ResourceType: Api, ResourceID: "api2", Action: ReadAPI}),
			),
			permissions: []string{"api.api1.read_api", "api.api1.update_api", "rbac.role1.read_role"},
			wantValid:   true,
		},
		{
			name: "Complex combination (Pass)",
			query: And(
				T(Tuple{ResourceType: Api, ResourceID: "api1", Action: ReadAPI}),
				Or(
					T(Tuple{ResourceType: Api, ResourceID: "api1", Action: UpdateAPI}),
					T(Tuple{ResourceType: Rbac, ResourceID: "role1", Action: ReadRole}),
				),
			),
			permissions: []string{"api.api1.read_api", "api.api1.update_api", "rbac.role1.read_role"},
			wantValid:   true,
		},
		{
			name:        "Asterisk permission literal match (Pass)",
			query:       S("api.*"),
			permissions: []string{"api.*", "api.read", "api.write"},
			wantValid:   true,
		},
		{
			name:        "Asterisk permission NOT wildcard (Fail)",
			query:       S("api.*"),
			permissions: []string{"api.read", "api.write", "api.delete"},
			wantValid:   false,
		},
		{
			name: "Complex query with asterisk permissions",
			query: Or(
				S("api.*"),
				S("api.read"),
			),
			permissions: []string{"api.read"},
			wantValid:   true,
		},
		{
			name:        "Permission with colon namespace (Pass)",
			query:       S("system:admin:read"),
			permissions: []string{"system:admin:read", "system:admin:write"},
			wantValid:   true,
		},
		{
			name:        "Permission with colon namespace (Fail)",
			query:       S("system:admin:write"),
			permissions: []string{"system:admin:read", "user:basic:read"},
			wantValid:   false,
		},
		{
			name: "Complex query with colons and other characters",
			query: And(
				S("system:admin:*"),
				Or(
					S("api_v2:read"),
					S("api-v2:write"),
				),
			),
			permissions: []string{"system:admin:*", "api_v2:read", "user:basic:read"},
			wantValid:   true,
		},
	}

	rbac := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := rbac.EvaluatePermissions(tt.query, tt.permissions)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if result.Valid != tt.wantValid {
				t.Errorf("want valid=%v, got valid=%v, message=%s",
					tt.wantValid, result.Valid, result.Message)
			}
		})
	}
}
