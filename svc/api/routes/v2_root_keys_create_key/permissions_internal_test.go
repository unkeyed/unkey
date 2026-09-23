package handler

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

func TestAuthorizePermissionsIndexesDistinctGrants(t *testing.T) {
	const count = 1000
	base := "unkey:v1:ws_one:projects/"
	p := &principal.Principal{AuthorizedWorkspaceID: "ws_one", Permissions: make([]string, count)}
	requested := make([]string, count)
	for i := range count {
		permission := fmt.Sprintf("%sproj_%04d#read", base, i)
		p.Permissions[i] = permission
		requested[count-1-i] = permission
	}

	got, err := authorizePermissions(t.Context(), p, requested)
	require.NoError(t, err)
	require.Len(t, got, count)
	require.Equal(t, base+"proj_0000#read", got[0])
	require.Equal(t, base+"proj_0999#read", got[count-1])
}

func TestAuthorizePermissionsRejectsMissingGrantAmongMaximumDistinctPermissions(t *testing.T) {
	const count = 1000
	base := "unkey:v1:ws_one:projects/"
	p := &principal.Principal{AuthorizedWorkspaceID: "ws_one", Permissions: make([]string, count)}
	requested := make([]string, count)
	for i := range count {
		p.Permissions[i] = fmt.Sprintf("%sproj_%04d#read", base, i)
		requested[i] = p.Permissions[i]
	}
	requested[count-1] = base + "missing#read"

	_, err := authorizePermissions(t.Context(), p, requested)
	require.Error(t, err)
	code, ok := fault.GetCode(err)
	require.True(t, ok)
	require.Equal(t, codes.Auth.Authorization.InsufficientPermissions.URN(), code)
}

func TestAuthorizePermissionsEnforcesContainmentBoundaries(t *testing.T) {
	base := "unkey:v1:ws_one:"
	tests := []struct {
		name      string
		caller    string
		requested string
		allowed   bool
	}{
		{"exact", base + "projects/Project_One#read", base + "projects/Project_One#read", true},
		{"collection wildcard", base + "projects/*#read", base + "projects/Project_One#read", true},
		{"subtree", base + "projects/Project_One/**#read", base + "projects/Project_One/keyspaces/ks_one#read", true},
		{"global action", base + "**#*", base + "projects/Project_One#delete", true},
		{"wildcard request contained", base + "projects/*/keyspaces/*#read", base + "projects/*/keyspaces/*#read", true},
		{"concrete does not contain wildcard request", base + "projects/Project_One#read", base + "projects/*#read", false},
		{"action boundary", base + "projects/*#read", base + "projects/Project_One#write", false},
		{"ancestor boundary", base + "projects/Project_One#read", base + "projects/Project_One/keyspaces/ks_one#read", false},
		{"case boundary", base + "projects/project_one#read", base + "projects/Project_One#read", false},
		{"workspace boundary", "unkey:v1:ws_two:**#*", base + "projects/Project_One#read", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &principal.Principal{AuthorizedWorkspaceID: "ws_one", Permissions: []string{tt.caller}}
			got, err := authorizePermissions(t.Context(), p, []string{tt.requested})
			if tt.allowed {
				require.NoError(t, err)
				require.Equal(t, []string{tt.requested}, got)
				return
			}
			require.Error(t, err)
			code, ok := fault.GetCode(err)
			require.True(t, ok)
			require.Equal(t, codes.Auth.Authorization.InsufficientPermissions.URN(), code)
		})
	}
}

func TestAuthorizePermissionsHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := authorizePermissions(ctx, &principal.Principal{AuthorizedWorkspaceID: "ws_one"}, []string{"unkey:v1:ws_one:projects/proj_one#read"})
	require.ErrorIs(t, err, context.Canceled)
}

func BenchmarkAuthorizePermissions1000Distinct(b *testing.B) {
	const count = 1000
	p := &principal.Principal{AuthorizedWorkspaceID: "ws_one", Permissions: make([]string, count)}
	requested := make([]string, count)
	for i := range count {
		permission := fmt.Sprintf("unkey:v1:ws_one:projects/proj_%04d/keyspaces/ks_%04d#read", i, i)
		p.Permissions[i] = fmt.Sprintf("unkey:v1:ws_one:projects/proj_%04d/keyspaces/*#read", i)
		requested[i] = permission
	}
	b.Run("authorized", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			_, err := authorizePermissions(context.Background(), p, requested)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("missing", func(b *testing.B) {
		requested[count-1] = "unkey:v1:ws_one:projects/missing#read"
		b.ReportAllocs()
		for range b.N {
			_, err := authorizePermissions(context.Background(), p, requested)
			if err == nil {
				b.Fatal("expected denial")
			}
		}
	})
}
