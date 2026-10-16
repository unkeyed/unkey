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

func TestMemberEmailsAndAdminEmails(t *testing.T) {
	orgID := uid.New(uid.TestPrefix)
	memberID := uid.New(uid.TestPrefix)
	adminID := uid.New(uid.TestPrefix)
	duplicateID := uid.New(uid.TestPrefix)
	emptyID := uid.New(uid.TestPrefix)
	cursor := uid.New(uid.TestPrefix)
	for _, adminsOnly := range []bool{false, true} {
		t.Run(map[bool]string{false: "members", true: "admins"}[adminsOnly], func(t *testing.T) {
			var cursors []string
			var users []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
				if r.URL.Path == "/user_management/organization_memberships" {
					assert.Equal(t, orgID, r.URL.Query().Get("organization_id"))
					assert.Equal(t, "active", r.URL.Query().Get("statuses"))
					assert.Equal(t, "100", r.URL.Query().Get("limit"))
					after := r.URL.Query().Get("after")
					cursors = append(cursors, after)
					page := membershipsPage{}
					if after == "" {
						page.Data = []membership{
							{UserID: memberID, Role: membershipRole{Slug: "member"}},
							{UserID: adminID, Role: membershipRole{Slug: "admin"}},
						}
						page.ListMetadata.After = cursor
					} else {
						assert.Equal(t, cursor, after)
						page.Data = []membership{
							{UserID: duplicateID, Role: membershipRole{Slug: "admin"}},
							{UserID: emptyID, Role: membershipRole{Slug: "admin"}},
						}
					}
					assert.NoError(t, json.NewEncoder(w).Encode(page))
					return
				}
				users = append(users, r.URL.Path)
				emails := map[string]string{
					memberID: "member@example.com", adminID: "admin@example.com",
					duplicateID: "admin@example.com", emptyID: "",
				}
				for id, email := range emails {
					if r.URL.Path == "/user_management/users/"+id {
						assert.NoError(t, json.NewEncoder(w).Encode(user{Email: email}))
						return
					}
				}
				http.NotFound(w, r)
			}))
			t.Cleanup(server.Close)
			client := New("test-key")
			client.baseURL = server.URL
			lookup := client.MemberEmails
			want := []string{"member@example.com", "admin@example.com"}
			if adminsOnly {
				lookup = client.AdminEmails
				want = []string{"admin@example.com", "admin@example.com"}
			}
			emails, err := lookup(t.Context(), orgID)
			require.NoError(t, err)
			require.Equal(t, want, emails)
			require.Equal(t, []string{"", cursor}, cursors)
			if adminsOnly {
				require.NotContains(t, users, "/user_management/users/"+memberID)
			}
		})
	}
}

func TestMemberEmailsPropagatesProviderErrors(t *testing.T) {
	for _, failure := range []string{"first page", "second page", "user"} {
		t.Run(failure, func(t *testing.T) {
			userID := uid.New(uid.TestPrefix)
			cursor := uid.New(uid.TestPrefix)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				isList := r.URL.Path == "/user_management/organization_memberships"
				if (failure == "first page" && isList) ||
					(failure == "second page" && r.URL.Query().Get("after") == cursor) ||
					(failure == "user" && !isList) {
					w.WriteHeader(http.StatusServiceUnavailable)
					return
				}
				if isList {
					assert.NoError(t, json.NewEncoder(w).Encode(membershipsPage{
						Data:         []membership{{UserID: userID, Role: membershipRole{Slug: "member"}}},
						ListMetadata: listMetadata{After: cursor},
					}))
					return
				}
				assert.NoError(t, json.NewEncoder(w).Encode(user{Email: "member@example.com"}))
			}))
			t.Cleanup(server.Close)
			client := New("test-key")
			client.baseURL = server.URL
			emails, err := client.MemberEmails(t.Context(), uid.New(uid.TestPrefix))
			require.Error(t, err)
			require.Nil(t, emails)
		})
	}
}
