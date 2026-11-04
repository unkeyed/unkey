package workos

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
)

// TestUserEmailTreatsMissingAsNoEmail guarantees that GDPR erasure can look up
// a user's email again after an earlier run deleted the user: a 404 means no
// email, while a server failure must surface so the lookup retries.
func TestUserEmailTreatsMissingAsNoEmail(t *testing.T) {
	userID := uid.New(uid.TestPrefix)
	address := userID + "@example.com"

	cases := []struct {
		code    int
		email   string
		success bool
	}{
		{code: http.StatusOK, email: address, success: true},
		{code: http.StatusNotFound, email: "", success: true},
		{code: http.StatusInternalServerError, email: "", success: false},
	}

	for _, tc := range cases {
		t.Run(http.StatusText(tc.code), func(t *testing.T) {
			var requests int
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/user_management/users/"+userID, r.URL.Path)
				assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
				w.WriteHeader(tc.code)
				if tc.code == http.StatusOK {
					assert.NoError(t, json.NewEncoder(w).Encode(user{Email: address}))
				}
			}))
			t.Cleanup(server.Close)

			email, err := New("test-key", server.URL).UserEmail(t.Context(), userID)
			require.Equal(t, 1, requests)
			require.Equal(t, tc.email, email)
			if tc.success {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
