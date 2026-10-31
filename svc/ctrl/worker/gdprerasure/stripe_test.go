package gdprerasure

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	stripe "github.com/stripe/stripe-go/v86"
	"github.com/unkeyed/unkey/pkg/uid"
)

// TestDeleteCustomerTreatsMissingAsDeleted guarantees that a workspace erasure
// converges on Stripe: a customer an earlier run already deleted counts as
// done, while any other Stripe failure surfaces so the step retries.
func TestDeleteCustomerTreatsMissingAsDeleted(t *testing.T) {
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
			body:    map[string]any{"object": "customer", "deleted": true},
			existed: true,
			success: true,
		},
		{
			name:    "already gone",
			status:  http.StatusNotFound,
			body:    map[string]any{"error": map[string]any{"type": "invalid_request_error", "code": "resource_missing", "message": "No such customer"}},
			existed: false,
			success: true,
		},
		{
			name:    "rejected",
			status:  http.StatusUnauthorized,
			body:    map[string]any{"error": map[string]any{"type": "invalid_request_error", "message": "Invalid API Key"}},
			existed: false,
			success: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			customerID := uid.New(uid.TestPrefix)
			var requests int
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				assert.Equal(t, http.MethodDelete, r.Method)
				assert.Equal(t, "/v1/customers/"+customerID, r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				assert.NoError(t, json.NewEncoder(w).Encode(tc.body))
			}))
			t.Cleanup(server.Close)

			customers := &stripeCustomers{client: stripe.NewClient("sk_test", stripe.WithBackends(stripe.NewBackendsWithConfig(&stripe.BackendConfig{
				URL:               stripe.String(server.URL),
				MaxNetworkRetries: stripe.Int64(0),
			})))}

			existed, err := customers.DeleteCustomer(t.Context(), customerID)
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
