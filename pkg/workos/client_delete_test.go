package workos

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
)

// TestDeleteTreatsMissingAsDeleted guarantees that GDPR erasure converges: a
// WorkOS record an earlier run already deleted answers 404, and that must count
// as done, while a server failure must surface so the step retries.
func TestDeleteTreatsMissingAsDeleted(t *testing.T) {
	userID := uid.New(uid.TestPrefix)
	orgID := uid.New(uid.OrgPrefix)

	operations := []struct {
		name   string
		path   string
		delete func(c Client, ctx context.Context) (bool, error)
	}{
		{
			name:   "user",
			path:   "/user_management/users/" + userID,
			delete: func(c Client, ctx context.Context) (bool, error) { return c.DeleteUser(ctx, userID) },
		},
		{
			name:   "organization",
			path:   "/organizations/" + orgID,
			delete: func(c Client, ctx context.Context) (bool, error) { return c.DeleteOrganization(ctx, orgID) },
		},
	}

	statuses := []struct {
		code    int
		success bool
		existed bool
	}{
		{code: http.StatusOK, success: true, existed: true},
		{code: http.StatusAccepted, success: true, existed: true},
		{code: http.StatusNoContent, success: true, existed: true},
		{code: http.StatusNotFound, success: true, existed: false},
		{code: http.StatusInternalServerError, success: false, existed: false},
	}

	for _, op := range operations {
		for _, status := range statuses {
			t.Run(op.name+"/"+http.StatusText(status.code), func(t *testing.T) {
				var requests int
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requests++
					assert.Equal(t, http.MethodDelete, r.Method)
					assert.Equal(t, op.path, r.URL.Path)
					assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
					w.WriteHeader(status.code)
				}))
				t.Cleanup(server.Close)

				existed, err := op.delete(New("test-key", server.URL), t.Context())
				require.Equal(t, 1, requests)
				require.Equal(t, status.existed, existed)
				if status.success {
					require.NoError(t, err)
				} else {
					require.Error(t, err)
				}
			})
		}
	}
}
