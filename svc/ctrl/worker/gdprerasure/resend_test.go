package gdprerasure

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
)

// TestDeleteContactTreatsMissingAsDeleted guarantees that a user erasure
// converges on Resend: a contact an earlier run already deleted counts as done,
// while any other Resend failure surfaces so the step retries.
func TestDeleteContactTreatsMissingAsDeleted(t *testing.T) {
	cases := []struct {
		name    string
		status  int
		body    map[string]any
		existed bool
		success bool
	}{
		{
			name:    "deleted",
			status:  http.StatusOK,
			body:    map[string]any{"object": "contact", "contact": uid.New(uid.TestPrefix), "deleted": true},
			existed: true,
			success: true,
		},
		{
			name:    "already gone",
			status:  http.StatusNotFound,
			body:    map[string]any{"statusCode": http.StatusNotFound, "name": "not_found", "message": "Contact not found"},
			existed: false,
			success: true,
		},
		{
			name:    "failed",
			status:  http.StatusInternalServerError,
			body:    map[string]any{"statusCode": http.StatusInternalServerError, "name": "application_error", "message": "Internal server error"},
			existed: false,
			success: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			email := uid.New(uid.TestPrefix) + "@example.com"
			var requests int
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				assert.Equal(t, http.MethodDelete, r.Method)
				assert.Equal(t, "/contacts/"+email, r.URL.Path)
				assert.Equal(t, "Bearer re_test", r.Header.Get("Authorization"))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				assert.NoError(t, json.NewEncoder(w).Encode(tc.body))
			}))
			t.Cleanup(server.Close)

			contacts := &resendContacts{apiKey: "re_test", baseURL: server.URL, client: server.Client()}

			existed, err := contacts.DeleteContact(t.Context(), email)
			require.Equal(t, 1, requests)
			require.Equal(t, tc.existed, existed)
			if tc.success {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
