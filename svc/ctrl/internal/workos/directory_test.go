package workos

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDirectoryMembers(t *testing.T) {
	t.Parallel()

	userRequests := make(map[string]int)
	var mu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch r.URL.Path {
		case "/user_management/organization_memberships":
			if r.URL.Query().Get("organization_id") != "org_1" || r.URL.Query().Get("statuses") != "active" || r.URL.Query().Get("limit") != "100" {
				t.Errorf("unexpected membership query: %q", r.URL.RawQuery)
			}
			if r.URL.Query().Get("after") == "" {
				writeDirectoryResponse(t, w, `{"data":[
					{"user_id":"admin","organization_id":"org_1","status":"active","role":{"slug":"admin"}},
					{"user_id":"inactive","organization_id":"org_1","status":"inactive","role":{"slug":"member"}},
					{"user_id":"cross-org","organization_id":"org_2","status":"active","role":{"slug":"admin"}},
					{"user_id":"unverified","organization_id":"org_1","status":"active","role":{"slug":"member"}}
				],"list_metadata":{"after":"page_2"}}`)
				return
			}
			writeDirectoryResponse(t, w, `{"data":[
				{"user_id":"member","organization_id":"org_1","status":"active","role":{"slug":"member"}},
				{"user_id":"admin","organization_id":"org_1","status":"active","role":{"slug":"member"}},
				{"user_id":"invalid","organization_id":"org_1","status":"active","role":{"slug":"member"}}
			],"list_metadata":{"after":""}}`)
		case "/user_management/users":
			if r.URL.Query().Get("organization_id") != "org_1" || r.URL.Query().Get("limit") != "100" {
				t.Errorf("unexpected user query: %q", r.URL.RawQuery)
			}
			userRequests["pages"]++
			if r.URL.Query().Get("after") == "" {
				writeDirectoryResponse(t, w, `{"data":[
					{"id":"admin","email":"admin@example.com","email_verified":true},
					{"id":"unverified","email":"unverified@example.com","email_verified":false},
					{"id":"cross-org","email":"foreign@example.com","email_verified":true}
				],"list_metadata":{"after":"users_2"}}`)
				return
			}
			writeDirectoryResponse(t, w, `{"data":[
				{"id":"member","email":" member@example.com ","email_verified":true},
				{"id":"inactive","email":"inactive@example.com","email_verified":true},
				{"id":"invalid","email":"Display Name <invalid@example.com>","email_verified":true}
			],"list_metadata":{}}`)
		default:
			userRequests[strings.TrimPrefix(r.URL.Path, "/user_management/users/")]++
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	directory := newTestDirectory(t, server, "secret")
	members, err := directory.Members(t.Context(), "org_1")
	require.NoError(t, err)
	require.Equal(t, []Member{
		{UserID: "admin", Email: "admin@example.com", Admin: true},
		{UserID: "member", Email: "member@example.com", Admin: false},
	}, members)
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
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tc.userError && r.URL.Path == "/user_management/organization_memberships" {
					writeDirectoryResponse(t, w, `{"data":[{"user_id":"user_1","organization_id":"org_1","status":"active","role":{"slug":"member"}}],"list_metadata":{}}`)
					return
				}
				w.WriteHeader(http.StatusUnauthorized)
				writeDirectoryResponse(t, w, `{"email":"private@example.com","credential":"secret"}`)
			}))
			t.Cleanup(server.Close)

			directory := newTestDirectory(t, server, "secret")
			_, err := directory.Members(t.Context(), "org_1")
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
		writeDirectoryResponse(t, w, fmt.Sprintf(`{"data":[],"list_metadata":{"after":%q}}`, "page_"+strconv.Itoa(int(page))))
	}))
	t.Cleanup(server.Close)

	directory := newTestDirectory(t, server, "secret")
	_, err := directory.Members(t.Context(), "org_1")
	require.ErrorIs(t, err, ErrDirectoryLimit)
	require.Equal(t, int32(maxDirectoryPages), requests.Load())
}

func TestDirectoryMembersBoundsMemberships(t *testing.T) {
	t.Parallel()

	for _, count := range []int{300, maxDirectoryMembers, maxDirectoryMembers + 1} {
		t.Run(strconv.Itoa(count), func(t *testing.T) {
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
				var rows []string
				switch r.URL.Path {
				case "/user_management/organization_memberships":
					for i := start; i < end; i++ {
						rows = append(rows, fmt.Sprintf(`{"user_id":"user_%d","organization_id":"org_1","status":"active","role":{"slug":"member"}}`, i))
					}
				case "/user_management/users":
					userRequests.Add(1)
					for i := start; i < end; i++ {
						rows = append(rows, fmt.Sprintf(`{"id":"user_%d","email":"user_%d@example.com","email_verified":true}`, i, i))
					}
				default:
					http.NotFound(w, r)
					return
				}
				after := ""
				if end < count {
					after = strconv.Itoa(end)
				}
				writeDirectoryResponse(t, w, fmt.Sprintf(`{"data":[%s],"list_metadata":{"after":%q}}`, strings.Join(rows, ","), after))
			}))
			t.Cleanup(server.Close)
			members, err := newTestDirectory(t, server, "secret").Members(t.Context(), "org_1")
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
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		switch r.URL.Path {
		case "/user_management/organization_memberships":
			if r.URL.Query().Get("organization_id") != "org_1" || r.URL.Query().Get("user_id") != "alice" || r.URL.Query().Get("statuses") != "active" {
				t.Error("explicit recipients must use a scoped membership lookup")
				http.Error(w, "full directory unavailable", http.StatusServiceUnavailable)
				return
			}
			writeDirectoryResponse(t, w, `{"data":[{"user_id":"alice","organization_id":"org_1","status":"active","role":{"slug":"admin"}}]}`)
		case "/user_management/users/alice":
			writeDirectoryResponse(t, w, `{"id":"alice","email":"alice@example.com","email_verified":true}`)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	members, err := newTestDirectory(t, server, "secret").LookupMembers(t.Context(), "org_1", []string{"alice", "alice"})
	require.NoError(t, err)
	require.Equal(t, []Member{{UserID: "alice", Email: "alice@example.com", Admin: true}}, members)
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
	type testCase struct {
		name, org, userID, status, email string
		verified, want                   bool
	}
	for _, tc := range []testCase{
		{name: "active", org: "org_1", userID: "alice", status: "active", email: "alice@example.com", verified: true, want: true},
		{name: "foreign org", org: "other", userID: "alice", status: "active", email: "alice@example.com", verified: true},
		{name: "other user", org: "org_1", userID: "bob", status: "active", email: "alice@example.com", verified: true},
		{name: "inactive", org: "org_1", userID: "alice", status: "inactive", email: "alice@example.com", verified: true},
		{name: "unverified", org: "org_1", userID: "alice", status: "active", email: "alice@example.com"},
		{name: "changed email", org: "org_1", userID: "alice", status: "active", email: "new@example.com", verified: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var userRequests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/user_management/organization_memberships" {
					if r.URL.Query().Get("user_id") != "alice" || r.URL.Query().Get("organization_id") != "org_1" {
						t.Error("missing membership filters")
					}
					writeDirectoryResponse(t, w, fmt.Sprintf(`{"data":[{"user_id":%q,"organization_id":%q,"status":%q}]}`, tc.userID, tc.org, tc.status))
				} else {
					userRequests.Add(1)
					writeDirectoryResponse(t, w, fmt.Sprintf(`{"email":%q,"email_verified":%t}`, tc.email, tc.verified))
				}
			}))
			t.Cleanup(server.Close)
			allowed, err := newTestDirectory(t, server, "secret").CanEmail(t.Context(), "org_1", "alice", "alice@example.com")
			require.NoError(t, err)
			require.Equal(t, tc.want, allowed)
			if tc.org != "org_1" || tc.userID != "alice" || tc.status != "active" {
				require.Zero(t, userRequests.Load())
			}
		})
	}
}

func writeDirectoryResponse(t *testing.T, w http.ResponseWriter, body string) {
	t.Helper()
	if _, err := io.WriteString(w, body); err != nil {
		t.Error(err)
	}
}
