package gdprerasure

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
)

// plainServer answers the client's GraphQL operations from fixed customers
// and records the customers it is asked to delete.
type plainServer struct {
	t            *testing.T
	byEmail      map[string]string
	byExternalID map[string]string
	errors       []plainError
	deleteError  *plainError
	deleted      []string
}

func (s *plainServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !assert.Equal(s.t, http.MethodPost, r.Method) || !assert.Equal(s.t, "Bearer plain_test", r.Header.Get("Authorization")) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var req plainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	res := plainResponse{Data: plainData{CustomerByEmail: nil, CustomerByExternalID: nil, DeleteCustomer: nil}, Errors: s.errors}
	switch {
	case len(s.errors) > 0:
	case strings.HasPrefix(req.Query, "query customerByEmail("):
		res.Data.CustomerByEmail = s.lookup(s.byEmail, req.Variables["email"])
	case strings.HasPrefix(req.Query, "query customerByExternalId("):
		res.Data.CustomerByExternalID = s.lookup(s.byExternalID, req.Variables["externalId"])
	case strings.HasPrefix(req.Query, "mutation deleteCustomer("):
		input, _ := req.Variables["input"].(map[string]any)
		customerID, _ := input["customerId"].(string)
		s.deleted = append(s.deleted, customerID)
		res.Data.DeleteCustomer = &plainDeleteOutput{Error: s.deleteError}
	default:
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(res)
}

func (s *plainServer) lookup(customers map[string]string, key any) *plainCustomer {
	k, _ := key.(string)
	id, ok := customers[k]
	if !ok {
		return nil
	}
	return &plainCustomer{ID: id}
}

// TestDeleteCustomersFindsByEmailOrExternalID guarantees that a user erasure
// deletes the user's Plain customer whether Plain knows it by the user's email
// or only by their WorkOS id, converges when there is none, and stops on any
// error Plain reports in the response body.
func TestDeleteCustomersFindsByEmailOrExternalID(t *testing.T) {
	email := uid.New(uid.TestPrefix) + "@example.com"
	userID := uid.New(uid.TestPrefix)
	customerID := uid.New(uid.TestPrefix)
	otherCustomerID := uid.New(uid.TestPrefix)

	cases := []struct {
		name         string
		email        string
		byEmail      map[string]string
		byExternalID map[string]string
		errors       []plainError
		deleteError  *plainError
		deleted      []string
		rows         int64
		success      bool
	}{
		{
			name:         "found by email",
			email:        email,
			byEmail:      map[string]string{email: customerID},
			byExternalID: nil,
			deleted:      []string{customerID},
			rows:         1,
			success:      true,
		},
		{
			name:         "found by email and external id",
			email:        email,
			byEmail:      map[string]string{email: customerID},
			byExternalID: map[string]string{userID: customerID},
			deleted:      []string{customerID},
			rows:         1,
			success:      true,
		},
		{
			name:         "found by external id",
			email:        "",
			byEmail:      nil,
			byExternalID: map[string]string{userID: customerID},
			deleted:      []string{customerID},
			rows:         1,
			success:      true,
		},
		{
			name:         "email and external id name different customers",
			email:        email,
			byEmail:      map[string]string{email: customerID},
			byExternalID: map[string]string{userID: otherCustomerID},
			deleted:      []string{customerID, otherCustomerID},
			rows:         2,
			success:      true,
		},
		{
			name:    "not found",
			email:   email,
			deleted: nil,
			rows:    0,
			success: true,
		},
		{
			name:    "graphql error",
			email:   email,
			byEmail: map[string]string{email: customerID},
			errors:  []plainError{{Message: "Unauthorized", Code: ""}},
			deleted: nil,
			rows:    0,
			success: false,
		},
		{
			name:        "mutation error",
			email:       email,
			byEmail:     map[string]string{email: customerID},
			deleteError: &plainError{Message: "Missing permission customer:delete", Code: "forbidden"},
			deleted:     []string{customerID},
			rows:        0,
			success:     false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fake := &plainServer{
				t:            t,
				byEmail:      tc.byEmail,
				byExternalID: tc.byExternalID,
				errors:       tc.errors,
				deleteError:  tc.deleteError,
				deleted:      nil,
			}
			server := httptest.NewServer(fake)
			t.Cleanup(server.Close)

			customers := &plainCustomers{apiKey: "plain_test", url: server.URL, client: server.Client()}

			rows, err := customers.DeleteCustomers(t.Context(), tc.email, userID)
			require.Equal(t, tc.deleted, fake.deleted)
			require.Equal(t, tc.rows, rows)
			if tc.success {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

// TestDeleteCustomersFailsOnHTTPError guarantees that a rejected request
// surfaces, so the step retries instead of reporting nothing to delete.
func TestDeleteCustomersFailsOnHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(plainResponse{
			Data:   plainData{CustomerByEmail: nil, CustomerByExternalID: nil, DeleteCustomer: nil},
			Errors: []plainError{{Message: "Authentication failed", Code: ""}},
		})
	}))
	t.Cleanup(server.Close)

	customers := &plainCustomers{apiKey: "plain_test", url: server.URL, client: server.Client()}

	rows, err := customers.DeleteCustomers(t.Context(), uid.New(uid.TestPrefix)+"@example.com", uid.New(uid.TestPrefix))
	require.ErrorContains(t, err, "status 401")
	require.Zero(t, rows)
}
