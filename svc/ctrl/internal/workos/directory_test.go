package workos

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestDirectoryMembers(t *testing.T) {
	t.Parallel()

	orgID := uid.New(uid.OrgPrefix)
	orgForeignID := uid.New(uid.OrgPrefix)
	adminID := uid.New(uid.TestPrefix)
	memberID := uid.New(uid.TestPrefix)
	inactiveID := uid.New(uid.TestPrefix)
	crossOrgID := uid.New(uid.TestPrefix)
	unverifiedID := uid.New(uid.TestPrefix)
	invalidID := uid.New(uid.TestPrefix)
	userRequests := make(map[string]int)
	var mu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch r.URL.Path {
		case "/user_management/organization_memberships":
			if r.URL.Query().Get("organization_id") != orgID || r.URL.Query().Get("statuses") != "active" || r.URL.Query().Get("limit") != "100" {
				t.Errorf("unexpected membership query: %q", r.URL.RawQuery)
			}
			if r.URL.Query().Get("after") == "" {
				writeDirectoryResponse(t, w, membershipsPage{
					Data: []membership{
						{UserID: adminID, OrganizationID: orgID, Status: "active", Role: membershipRole{Slug: "admin"}},
						{UserID: inactiveID, OrganizationID: orgID, Status: "inactive", Role: membershipRole{Slug: "member"}},
						{UserID: crossOrgID, OrganizationID: orgForeignID, Status: "active", Role: membershipRole{Slug: "admin"}},
						{UserID: unverifiedID, OrganizationID: orgID, Status: "active", Role: membershipRole{Slug: "member"}},
					},
					ListMetadata: listMetadata{After: "page_2"},
				})
				return
			}
			writeDirectoryResponse(t, w, membershipsPage{
				Data: []membership{
					{UserID: memberID, OrganizationID: orgID, Status: "active", Role: membershipRole{Slug: "member"}},
					{UserID: adminID, OrganizationID: orgID, Status: "active", Role: membershipRole{Slug: "member"}},
					{UserID: invalidID, OrganizationID: orgID, Status: "active", Role: membershipRole{Slug: "member"}},
				},
			})
		case "/user_management/users":
			if r.URL.Query().Get("organization_id") != orgID || r.URL.Query().Get("limit") != "100" {
				t.Errorf("unexpected user query: %q", r.URL.RawQuery)
			}
			userRequests["pages"]++
			if r.URL.Query().Get("after") == "" {
				writeDirectoryResponse(t, w, usersPage{
					Data: []user{
						{ID: adminID, Email: "admin@example.com", EmailVerified: true},
						{ID: unverifiedID, Email: "unverified@example.com", EmailVerified: false},
						{ID: crossOrgID, Email: "foreign@example.com", EmailVerified: true},
					},
					ListMetadata: listMetadata{After: "users_2"},
				})
				return
			}
			writeDirectoryResponse(t, w, usersPage{
				Data: []user{
					{ID: memberID, Email: " member@example.com ", EmailVerified: true},
					{ID: inactiveID, Email: "inactive@example.com", EmailVerified: true},
					{ID: invalidID, Email: "Display Name <invalid@example.com>", EmailVerified: true},
				},
			})
		default:
			userRequests[strings.TrimPrefix(r.URL.Path, "/user_management/users/")]++
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	directory := newTestDirectory(t, server, "secret")
	members, err := directory.Members(t.Context(), orgID)
	require.NoError(t, err)
	want := []Member{
		{UserID: adminID, Email: "admin@example.com", Admin: true},
		{UserID: memberID, Email: "member@example.com", Admin: false},
	}
	slices.SortFunc(want, func(a, b Member) int { return strings.Compare(a.UserID, b.UserID) })
	require.Equal(t, want, members)
	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, map[string]int{"pages": 2}, userRequests)
}

func TestNewDirectoryRejectsEmptyAPIKey(t *testing.T) {
	t.Parallel()

	directory, err := NewDirectory(" \t")
	require.Nil(t, directory)
	require.EqualError(t, err, "workos API key is not configured")
}

func TestDirectoryMembersSanitizesProviderErrors(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name      string
		userError bool
		want      string
	}
	for _, tc := range []testCase{
		{name: "membership", want: "list workos organization memberships"},
		{name: "user", userError: true, want: "list workos organization users"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			orgID := uid.New(uid.OrgPrefix)
			userID := uid.New(uid.TestPrefix)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tc.userError && r.URL.Path == "/user_management/organization_memberships" {
					writeDirectoryResponse(t, w, membershipsPage{
						Data: []membership{{UserID: userID, OrganizationID: orgID, Status: "active", Role: membershipRole{Slug: "member"}}},
					})
					return
				}
				w.WriteHeader(http.StatusUnauthorized)
				writeDirectoryResponse(t, w, struct {
					Email      string `json:"email"`
					Credential string `json:"credential"`
				}{
					Email:      "private@example.com",
					Credential: "secret",
				})
			}))
			t.Cleanup(server.Close)

			directory := newTestDirectory(t, server, "secret")
			_, err := directory.Members(t.Context(), orgID)
			require.EqualError(t, err, tc.want)
			require.NotContains(t, err.Error(), "private@example.com")
			require.NotContains(t, err.Error(), "secret")
		})
	}
}

func TestDirectoryMembersBoundsPages(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		page := requests.Add(1)
		writeDirectoryResponse(t, w, membershipsPage{
			Data:         []membership{},
			ListMetadata: listMetadata{After: "page_" + strconv.Itoa(int(page))},
		})
	}))
	t.Cleanup(server.Close)

	directory := newTestDirectory(t, server, "secret")
	_, err := directory.Members(t.Context(), uid.New(uid.OrgPrefix))
	require.ErrorIs(t, err, ErrDirectoryLimit)
	require.Equal(t, int32(maxDirectoryPages), requests.Load())
}

func TestDirectoryMembersBoundsMemberships(t *testing.T) {
	t.Parallel()

	for _, count := range []int{300, maxDirectoryMembers, maxDirectoryMembers + 1} {
		t.Run(strconv.Itoa(count), func(t *testing.T) {
			orgID := uid.New(uid.OrgPrefix)
			memberships := make([]membership, count)
			users := make([]user, count)
			for i := range count {
				userID := uid.New(uid.TestPrefix)
				memberships[i] = membership{
					UserID:         userID,
					OrganizationID: orgID,
					Status:         "active",
					Role:           membershipRole{Slug: "member"},
				}
				users[i] = user{
					ID:            userID,
					Email:         fmt.Sprintf("user_%d@example.com", i),
					EmailVerified: true,
				}
			}
			var requests, userRequests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				start := 0
				if after := r.URL.Query().Get("after"); after != "" {
					var err error
					start, err = strconv.Atoi(after)
					if err != nil {
						t.Error(err)
						http.Error(w, "invalid cursor", http.StatusBadRequest)
						return
					}
				}
				end := min(start+membershipPageSize, count)
				after := ""
				if end < count {
					after = strconv.Itoa(end)
				}
				switch r.URL.Path {
				case "/user_management/organization_memberships":
					writeDirectoryResponse(t, w, membershipsPage{
						Data:         memberships[start:end],
						ListMetadata: listMetadata{After: after},
					})
				case "/user_management/users":
					userRequests.Add(1)
					writeDirectoryResponse(t, w, usersPage{
						Data:         users[start:end],
						ListMetadata: listMetadata{After: after},
					})
				default:
					http.NotFound(w, r)
				}
			}))
			t.Cleanup(server.Close)
			members, err := newTestDirectory(t, server, "secret").Members(t.Context(), orgID)
			if count > maxDirectoryMembers {
				require.ErrorIs(t, err, ErrDirectoryLimit)
				require.Empty(t, members)
				require.Zero(t, userRequests.Load())
			} else {
				require.NoError(t, err)
				require.Len(t, members, count)
				require.Equal(t, int32(2*((count+membershipPageSize-1)/membershipPageSize)), requests.Load())
			}
		})
	}
}

func TestDirectoryLookupMembersSkipsFullDirectory(t *testing.T) {
	orgID := uid.New(uid.OrgPrefix)
	userID := uid.New(uid.TestPrefix)
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		switch r.URL.Path {
		case "/user_management/organization_memberships":
			if r.URL.Query().Get("organization_id") != orgID || r.URL.Query().Get("user_id") != userID || r.URL.Query().Get("statuses") != "active" {
				t.Error("explicit recipients must use a scoped membership lookup")
				http.Error(w, "full directory unavailable", http.StatusServiceUnavailable)
				return
			}
			writeDirectoryResponse(t, w, membershipsPage{
				Data: []membership{{UserID: userID, OrganizationID: orgID, Status: "active", Role: membershipRole{Slug: "admin"}}},
			})
		case "/user_management/users/" + userID:
			writeDirectoryResponse(t, w, user{ID: userID, Email: "alice@example.com", EmailVerified: true})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	members, err := newTestDirectory(t, server, "secret").LookupMembers(t.Context(), orgID, []string{userID, userID})
	require.NoError(t, err)
	require.Equal(t, []Member{{UserID: userID, Email: "alice@example.com", Admin: true}}, members)
	require.Equal(t, int32(2), requests.Load())
}

func newTestDirectory(t *testing.T, server *httptest.Server, apiKey string) Directory {
	t.Helper()

	directory, err := NewDirectory(apiKey)
	require.NoError(t, err)
	workosClient := directory.(*client)
	workosClient.baseURL = server.URL
	workosClient.http = server.Client()
	return directory
}

func TestDirectoryCanEmail(t *testing.T) {
	orgID := uid.New(uid.OrgPrefix)
	userID := uid.New(uid.TestPrefix)
	type testCase struct {
		name, org, userID, status, email string
		verified, want                   bool
	}
	for _, tc := range []testCase{
		{name: "active", org: orgID, userID: userID, status: "active", email: "alice@example.com", verified: true, want: true},
		{name: "foreign org", org: uid.New(uid.OrgPrefix), userID: userID, status: "active", email: "alice@example.com", verified: true},
		{name: "other user", org: orgID, userID: uid.New(uid.TestPrefix), status: "active", email: "alice@example.com", verified: true},
		{name: "inactive", org: orgID, userID: userID, status: "inactive", email: "alice@example.com", verified: true},
		{name: "unverified", org: orgID, userID: userID, status: "active", email: "alice@example.com"},
		{name: "changed email", org: orgID, userID: userID, status: "active", email: "new@example.com", verified: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var userRequests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/user_management/organization_memberships" {
					if r.URL.Query().Get("user_id") != userID || r.URL.Query().Get("organization_id") != orgID {
						t.Error("missing membership filters")
					}
					writeDirectoryResponse(t, w, membershipsPage{
						Data: []membership{{UserID: tc.userID, OrganizationID: tc.org, Status: tc.status}},
					})
				} else {
					userRequests.Add(1)
					writeDirectoryResponse(t, w, user{ID: userID, Email: tc.email, EmailVerified: tc.verified})
				}
			}))
			t.Cleanup(server.Close)
			allowed, err := newTestDirectory(t, server, "secret").CanEmail(t.Context(), orgID, userID, "alice@example.com")
			require.NoError(t, err)
			require.Equal(t, tc.want, allowed)
			if tc.org != orgID || tc.userID != userID || tc.status != "active" {
				require.Zero(t, userRequests.Load())
			}
		})
	}
}

func writeDirectoryResponse(t *testing.T, w http.ResponseWriter, body any) {
	t.Helper()
	if err := json.NewEncoder(w).Encode(body); err != nil {
		t.Error(err)
	}
}
