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
