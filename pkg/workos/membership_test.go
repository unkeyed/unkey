package workos

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestActiveRolesReadsMembershipAndReportsRemoval(t *testing.T) {
	orgID := uid.New(uid.OrgPrefix)
	userID := uid.New(uid.TestPrefix)
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		require.Equal(t, "Bearer sk_test", r.Header.Get("Authorization"))
		require.Equal(t, orgID, r.URL.Query().Get("organization_id"))
		require.Equal(t, userID, r.URL.Query().Get("user_id"))
		require.Equal(t, "active", r.URL.Query().Get("statuses"))
		if calls == 1 {
			_, _ = w.Write([]byte(`{"data":[{"role":{"slug":"developer"},"roles":[{"slug":"developer"},{"slug":"viewer"}]}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	t.Cleanup(srv.Close)

	client := New("sk_test", srv.URL)
	roles, found, err := client.ActiveRoles(t.Context(), orgID, userID)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, []string{"developer", "viewer"}, roles)

	roles, found, err = client.ActiveRoles(t.Context(), orgID, userID)
	require.NoError(t, err)
	require.False(t, found)
	require.Empty(t, roles)
}
