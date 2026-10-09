package workos

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApplyPermissionCeiling(t *testing.T) {
	t.Parallel()

	const workspaceID = "ws_123"
	computeReadWrite := []string{"projects/*#read", "projects/*#write"}

	tests := []struct {
		name     string
		roles    []string
		granted  []string
		ceiling  []string
		want     []string
		wantNone bool
	}{
		{
			name:    "admin intersect compute cap",
			roles:   []string{"admin"},
			ceiling: ComputePermissionCeiling,
		},
		{
			name:    "developer intersect compute cap",
			roles:   []string{"developer"},
			ceiling: ComputePermissionCeiling,
		},
		{
			name:    "admin narrows wildcard to the cap",
			roles:   []string{"admin"},
			ceiling: computeReadWrite,
			want: []string{
				"unkey:v1:ws_123:projects/*#read",
				"unkey:v1:ws_123:projects/*#write",
			},
		},
		{
			name:    "developer keeps permissions inside the cap",
			roles:   []string{"developer"},
			ceiling: computeReadWrite,
			want: []string{
				"unkey:v1:ws_123:projects/*#read",
				"unkey:v1:ws_123:projects/*#write",
			},
		},
		{
			name:    "viewer intersect compute cap keeps read only",
			roles:   []string{"viewer"},
			ceiling: computeReadWrite,
			want:    []string{"unkey:v1:ws_123:projects/*#read"},
		},
		{
			name:     "no overlap yields none",
			granted:  []string{"unkey:v1:ws_123:projects/*/identities/*#read"},
			ceiling:  []string{"projects/*#read"},
			wantNone: true,
		},
		{
			name:     "unknown role yields none",
			roles:    []string{"not-a-role"},
			ceiling:  ComputePermissionCeiling,
			wantNone: true,
		},
		{
			name:    "narrower role permission is kept",
			granted: []string{"unkey:v1:ws_123:projects/proj_1#read"},
			ceiling: []string{"projects/*#read", "projects/*#write"},
			want:    []string{"unkey:v1:ws_123:projects/proj_1#read"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			granted := tt.granted
			if tt.roles != nil {
				granted = permissionsForRoles(workspaceID, tt.roles)
			}
			got, err := applyPermissionCeiling(workspaceID, granted, tt.ceiling)
			require.NoError(t, err)
			if tt.wantNone {
				require.Empty(t, got)
				return
			}
			if tt.name == "admin intersect compute cap" || tt.name == "developer intersect compute cap" {
				require.Equal(t, scopedCeiling(workspaceID, ComputePermissionCeiling), got)
				require.NotContains(t, got, "unkey:v1:ws_123:**#*")
				require.NotContains(t, got, "unkey:v1:ws_123:projects/*/keyspaces/*#read")
				return
			}
			require.Equal(t, tt.want, got)
		})
	}
}

// TestApplyPermissionCeiling_ViewerCannotReadSecretsBeyondRead grants the
// viewer role only the read half of the Compute ceiling, including variables
// read, and never a write the role does not have.
func TestApplyPermissionCeiling_ViewerCannotReadSecretsBeyondRead(t *testing.T) {
	t.Parallel()

	got, err := applyPermissionCeiling("ws_123", permissionsForRoles("ws_123", []string{"viewer"}), ComputePermissionCeiling)
	require.NoError(t, err)
	require.Contains(t, got, "unkey:v1:ws_123:projects/*/apps/*/environments/*/variables/*#read")
	require.NotContains(t, got, "unkey:v1:ws_123:projects/*/apps/*/environments/*/variables/*#write")
	require.NotContains(t, got, "unkey:v1:ws_123:projects/*#write")
	for _, permission := range got {
		require.Contains(t, permission, "#read")
	}
}

// TestApplyPermissionCeiling_NilCeilingKeepsRolePermissions guarantees omitting
// the ceiling does not narrow dashboard role mapping.
func TestApplyPermissionCeiling_NilCeilingKeepsRolePermissions(t *testing.T) {
	t.Parallel()

	granted := permissionsForRoles("ws_123", []string{"admin"})
	got, err := applyPermissionCeiling("ws_123", granted, nil)
	require.NoError(t, err)
	require.Equal(t, granted, got)
}

// TestValidatePermissionCeiling guarantees the Compute preset parses and a
// malformed pattern does not.
func TestValidatePermissionCeiling(t *testing.T) {
	t.Parallel()

	require.NoError(t, ValidatePermissionCeiling(nil))
	require.NoError(t, ValidatePermissionCeiling(ComputePermissionCeiling))
	require.Error(t, ValidatePermissionCeiling([]string{"not-a-permission"}))
	require.Error(t, ValidatePermissionCeiling([]string{"projects/*#rotate"}))
	require.Error(t, ValidatePermissionCeiling([]string{"projects/*#*"}))
}

func scopedCeiling(workspaceID string, patterns []string) []string {
	out := make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		perm, err := parseCeilingPattern(workspaceID, pattern)
		if err != nil {
			panic(err)
		}
		out = append(out, perm.value)
	}
	return out
}
