package workos

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestAdminEmailsRetriesUnavailableEndpoint(t *testing.T) {
	userID := uid.New(uid.TestPrefix)
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if requests.Add(1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		var response any = user{Email: "admin@example.com"}
		if r.URL.Path == "/user_management/organization_memberships" {
			response = membershipsPage{
				Data:         []membership{{UserID: userID, Role: membershipRole{Slug: adminRoleSlug}}},
				ListMetadata: listMetadata{After: ""},
			}
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	resolver := New("test-key").(*client)
	resolver.baseURL = server.URL
	t.Cleanup(resolver.http.CloseIdleConnections)
	emails, err := resolver.AdminEmails(context.Background(), uid.New(uid.OrgPrefix))
	require.NoError(t, err)
	require.Equal(t, []string{"admin@example.com"}, emails)
}
