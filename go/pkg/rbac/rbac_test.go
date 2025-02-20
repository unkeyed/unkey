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
			name:        "Simple role check (Pass)",
			query:       P("admin"),
			permissions: []string{"admin", "user", "guest"},
			wantValid:   true,
		},
		{
			name:        "Simple role check (Fail)",
			query:       P("developer"),
			permissions: []string{"admin", "user", "guest"},
			wantValid:   false,
		},
		{
			name: "AND of two permissions (Pass)",
			query: And(
				P("admin"),
				P("user"),
			),
			permissions: []string{"admin", "user", "guest"},
			wantValid:   true,
		},
		{
			name: "OR of two permissions (Pass)",
			query: Or(
				P("admin"),
				P("developer"),
			),
			permissions: []string{"admin", "user", "guest"},
			wantValid:   true,
		},
		{
			name: "Complex combination (Pass)",
			query: And(
				P("admin"),
				Or(
					P("user"),
					P("guest"),
				),
			),
			permissions: []string{"admin", "user", "guest"},
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
