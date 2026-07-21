package handler

import (
	"fmt"
	"net/http"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
)

func TestKeySpaceAuthorizationWorkLimit(t *testing.T) {
	tests := []struct {
		name    string
		ids     []string
		wantErr bool
	}{
		{name: "zero", ids: nil},
		{name: "one", ids: []string{"ks_1"}},
		{name: "ten", ids: []string{"ks_1", "ks_2", "ks_3", "ks_4", "ks_5", "ks_6", "ks_7", "ks_8", "ks_9", "ks_10"}},
		{name: "eleven", ids: []string{"ks_1", "ks_2", "ks_3", "ks_4", "ks_5", "ks_6", "ks_7", "ks_8", "ks_9", "ks_10", "ks_11"}, wantErr: true},
		{name: "duplicates count once", ids: slices.Concat([]string{"ks_1", "ks_2", "ks_3", "ks_4", "ks_5", "ks_6", "ks_7", "ks_8", "ks_9", "ks_10"}, []string{"ks_1", "ks_2", "ks_3", "ks_4", "ks_5", "ks_6", "ks_7", "ks_8", "ks_9", "ks_10"})},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Security guarantee: authorization fan-out is capped by unique key-space IDs.
			err := validateKeySpaceAuthorizationWork(tt.ids)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func Test403_NoAnalyticsPermission(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})

	workspace := h.CreateWorkspace()
	_ = h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})
	h.SetupAnalytics(workspace.ID)

	// Create root key WITHOUT read_analytics permission
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_api")

	route := &Handler{
		DB:                         h.DB,
		AnalyticsConnectionManager: h.AnalyticsConnectionManager,
		Caches:                     h.Caches,
	}
	h.Register(route)

	headers := http.Header{
		"Authorization": []string{"Bearer " + rootKey},
		"Content-Type":  []string{"application/json"},
	}

	req := Request{
		Query: "SELECT COUNT(*) FROM key_verifications_v1",
	}

	res := testutil.CallRoute[Request, Response](h, route, headers, req)
	require.Equal(t, 403, res.Status)
}

func Test403_WrongApiPermission(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})

	workspace := h.CreateWorkspace()
	api1 := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})
	api2 := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})
	h.SetupAnalytics(workspace.ID)

	// Create root key with permission only for api1
	rootKey := h.CreateRootKey(workspace.ID, "api."+api1.ID+".read_analytics")

	route := &Handler{
		DB:                         h.DB,
		AnalyticsConnectionManager: h.AnalyticsConnectionManager,
		Caches:                     h.Caches,
	}
	h.Register(route)

	headers := http.Header{
		"Authorization": []string{"Bearer " + rootKey},
		"Content-Type":  []string{"application/json"},
	}

	// Query filtering by api2's key_space_id but user only has permission for api1
	req := Request{
		Query: fmt.Sprintf("SELECT COUNT(*) FROM key_verifications_v1 WHERE key_space_id = '%s'", api2.KeyAuthID.String),
	}

	res := testutil.CallRoute[Request, Response](h, route, headers, req)
	require.Equal(t, 403, res.Status)
}

func Test403_ScopedPermissionRequiresInjectedSecurityFilter(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})
	workspace := h.CreateWorkspace()
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	h.SetupAnalytics(workspace.ID)
	rootKey := h.CreateRootKey(workspace.ID, "api."+api.ID+".read_analytics")
	route := &Handler{DB: h.DB, AnalyticsConnectionManager: h.AnalyticsConnectionManager, Caches: h.Caches}
	h.Register(route)

	// Security guarantee: scoped access fails closed when the query shape prevents RLS injection.
	res := testutil.CallRoute[Request, Response](h, route, http.Header{
		"Authorization": []string{"Bearer " + rootKey},
		"Content-Type":  []string{"application/json"},
	}, Request{Query: "SELECT COUNT(*) FROM key_verifications_v1 AS v CROSS JOIN (SELECT 1) AS x"})

	require.Equal(t, http.StatusForbidden, res.Status)
}
