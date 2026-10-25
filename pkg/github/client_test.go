package github

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clock"
)

func TestGetInstallationRepoMembership(t *testing.T) {
	member := ghRepoResponse{ID: 42, FullName: "Acme/App", DefaultBranch: "develop", Private: true}
	public := ghRepoResponse{ID: 43, FullName: "other/app", DefaultBranch: "main", Private: false}
	pageFull := make([]ghRepoResponse, 100)
	for i := range pageFull {
		pageFull[i] = public
	}
	tests := []struct {
		name      string
		repo      string
		pages     [][]ghRepoResponse
		status    int
		malformed bool
		transport bool
		want      *RepoInfo
		wantErr   bool
	}{
		{name: "public nonmember is not authorized by public metadata", repo: public.FullName, pages: [][]ghRepoResponse{{member}}},
		{name: "same bare name under another owner is not a match", repo: member.FullName, pages: [][]ghRepoResponse{{public}}},
		{name: "private member beyond first page matches case insensitively", repo: "acme/app", pages: [][]ghRepoResponse{pageFull, {member}}, want: &RepoInfo{ID: member.ID, FullName: member.FullName, DefaultBranch: member.DefaultBranch, Private: true}},
		{name: "public member retains canonical metadata", repo: "OTHER/APP", pages: [][]ghRepoResponse{{public}}, want: &RepoInfo{ID: public.ID, FullName: public.FullName, DefaultBranch: public.DefaultBranch, Private: false}},
		{name: "absence after full page", repo: member.FullName, pages: [][]ghRepoResponse{pageFull, {}}},
		{name: "rate limit is an error", repo: member.FullName, status: http.StatusForbidden, wantErr: true},
		{name: "upstream failure is an error", repo: member.FullName, status: http.StatusBadGateway, wantErr: true},
		{name: "installation endpoint not found is an error", repo: member.FullName, status: http.StatusNotFound, wantErr: true},
		{name: "malformed response is an error", repo: member.FullName, malformed: true, wantErr: true},
		{name: "transport failure is an error", repo: member.FullName, transport: true, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := cache.New(cache.Config[int64, InstallationToken]{
				Fresh: time.Hour, Stale: time.Hour, MaxSize: 100, Resource: "test_github_token", Clock: clock.New(),
			})
			require.NoError(t, err)
			t.Cleanup(tokens.Close)
			const installationID int64 = 123
			tokens.Set(t.Context(), installationID, InstallationToken{Token: "installation-token"})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/installation/repositories" {
					if err := json.NewEncoder(w).Encode(public); err != nil {
						t.Error(err)
					}
					return
				}
				if tt.status != 0 {
					w.WriteHeader(tt.status)
					return
				}
				if tt.malformed {
					if _, err := w.Write([]byte("{")); err != nil {
						t.Error(err)
					}
					return
				}
				page, err := strconv.Atoi(r.URL.Query().Get("page"))
				if err != nil || page < 1 || page > len(tt.pages) {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				response := ghInstallationReposResponse{Repositories: tt.pages[page-1]}
				if err := json.NewEncoder(w).Encode(response); err != nil {
					t.Error(err)
				}
			}))
			t.Cleanup(server.Close)
			if tt.transport {
				server.Close()
			}
			client := &Client{baseURL: server.URL, httpClient: server.Client(), tokenCache: tokens}
			info, err := client.GetInstallationRepo(installationID, tt.repo)
			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, info)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, info)
			}
		})
	}
}

func TestResolveCommitAuthor(t *testing.T) {
	tests := []struct {
		name              string
		commit            ghCommitResponse
		expectedHandle    string
		expectedAvatarURL string
	}{
		{
			name: "unverified commit with misconfigured email falls back to git metadata",
			commit: ghCommitResponse{
				SHA: "aaa1111111111111111111111111111111111111",
				Commit: ghCommitDetail{
					Message: "fix: update config",
					Author: ghCommitAuthor{
						Name:  "alice",
						Email: "alice@alices-MacBook-Pro.local",
						Date:  "2026-04-22T13:09:45Z",
					},
					Verification: ghCommitVerification{Verified: false},
				},
				Author: ghUser{
					Login:     "wronguser",
					AvatarURL: "https://avatars.githubusercontent.com/u/99999?v=4",
				},
			},
			expectedHandle:    "alice",
			expectedAvatarURL: "https://www.gravatar.com/avatar/b91be39b212b63b1fac5e8041f01ae89?d=identicon",
		},
		{
			name: "verified commit uses GitHub resolved author",
			commit: ghCommitResponse{
				SHA: "bbb2222222222222222222222222222222222222",
				Commit: ghCommitDetail{
					Message: "Update README.md",
					Author: ghCommitAuthor{
						Name:  "Alice",
						Email: "alice@example.com",
						Date:  "2026-04-22T13:50:03Z",
					},
					Verification: ghCommitVerification{Verified: true},
				},
				Author: ghUser{
					Login:     "alice-gh",
					AvatarURL: "https://avatars.githubusercontent.com/u/12345?v=4",
				},
			},
			expectedHandle:    "alice-gh",
			expectedAvatarURL: "https://avatars.githubusercontent.com/u/12345?v=4",
		},
		{
			name: "verified commit from local with proper config uses GitHub resolved author",
			commit: ghCommitResponse{
				SHA: "ccc3333333333333333333333333333333333333",
				Commit: ghCommitDetail{
					Message: "feat: add endpoint",
					Author: ghCommitAuthor{
						Name:  "bob",
						Email: "bob@company.com",
						Date:  "2026-04-22T13:51:08Z",
					},
					Verification: ghCommitVerification{Verified: true},
				},
				Author: ghUser{
					Login:     "bob-dev",
					AvatarURL: "https://avatars.githubusercontent.com/u/67890?v=4",
				},
			},
			expectedHandle:    "bob-dev",
			expectedAvatarURL: "https://avatars.githubusercontent.com/u/67890?v=4",
		},
		{
			name: "verified commit but empty GitHub login falls back to gravatar",
			commit: ghCommitResponse{
				SHA: "ddd4444444444444444444444444444444444444",
				Commit: ghCommitDetail{
					Message: "chore: cleanup",
					Author: ghCommitAuthor{
						Name:  "bob",
						Email: "bob@localhost",
						Date:  "2026-04-22T15:00:00Z",
					},
					Verification: ghCommitVerification{Verified: true},
				},
				Author: ghUser{},
			},
			expectedHandle:    "bob",
			expectedAvatarURL: "https://www.gravatar.com/avatar/a1d4544a85858c1d7562c5659166af24?d=identicon",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handle, avatarURL := resolveCommitAuthor(tt.commit)
			require.Equal(t, tt.expectedHandle, handle)
			require.Equal(t, tt.expectedAvatarURL, avatarURL)
		})
	}
}

// TestProbeRepoVisibility pins the status mapping: only 200 and 404 are
// answers. A rate limit (403) must surface as an error, never as "private",
// so callers can distinguish an inconclusive probe from a real answer.
func TestProbeRepoVisibility(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		wantPublic bool
		wantErr    bool
	}{
		{name: "200 means public", status: http.StatusOK, wantPublic: true},
		{name: "404 means private", status: http.StatusNotFound, wantPublic: false},
		{name: "403 rate limit is an error, not private", status: http.StatusForbidden, wantErr: true},
		{name: "500 is an error", status: http.StatusInternalServerError, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
			}))
			defer srv.Close()

			public, err := probeRepoVisibility(srv.Client(), srv.URL, "acme/app")
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantPublic, public)
		})
	}
}

// TestGetRawFile pins that a missing file is not an error: most repositories
// have no .gitmodules, and treating 404 as a failure would break their builds.
func TestGetRawFile(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		body      string
		wantFound bool
		wantErr   bool
	}{
		{name: "200 returns the body", status: http.StatusOK, body: "[submodule \"lib\"]\n", wantFound: true},
		{name: "404 means not found", status: http.StatusNotFound, body: "{}", wantFound: false},
		{name: "403 is an error", status: http.StatusForbidden, body: "{}", wantErr: true},
		{name: "body over the limit is an error", status: http.StatusOK, body: strings.Repeat("a", maxFileContentBytes+1), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			content, found, err := getRawFile(srv.Client(), srv.URL, nil)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantFound, found)
			if tt.wantFound {
				require.Equal(t, tt.body, string(content))
			}
		})
	}
}

func TestFindBotCommentBeyondFirstPage(t *testing.T) {
	const marker = "<!-- unkey-deploy -->"
	for _, scenario := range []string{"found", "absent", "page_failed"} {
		t.Run(scenario, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				var comments []ghIssueComment
				switch req.URL.Query().Get("page") {
				case "", "1":
					comments = make([]ghIssueComment, 100)
					for i := range comments {
						comments[i] = ghIssueComment{ID: int64(i + 1), Body: "unrelated"}
					}
				case "2":
					if scenario == "page_failed" {
						w.WriteHeader(http.StatusServiceUnavailable)
						return
					}
					comments = []ghIssueComment{{ID: 101, Body: "unrelated"}}
					if scenario == "found" {
						comments = append(comments, ghIssueComment{ID: 102, Body: marker + "existing report"})
					}
				default:
					http.Error(w, "unexpected page", http.StatusBadRequest)
					return
				}
				if err := json.NewEncoder(w).Encode(comments); err != nil {
					t.Error(err)
				}
			}))
			t.Cleanup(server.Close)
			tokens, err := cache.New(cache.Config[int64, InstallationToken]{
				Fresh: time.Hour, Stale: time.Hour, MaxSize: 10, Resource: "github_test_tokens", Clock: clock.New(),
			})
			require.NoError(t, err)
			t.Cleanup(tokens.Close)
			tokens.Set(t.Context(), 1, InstallationToken{Token: "test-token"})
			client := &Client{baseURL: server.URL, httpClient: server.Client(), tokenCache: tokens}
			id, body, err := client.FindBotComment(1, "acme/app", 23, marker)
			if scenario == "page_failed" {
				require.Error(t, err, "incomplete search must not authorize a new comment")
				return
			}
			require.NoError(t, err)
			if scenario == "found" {
				require.EqualValues(t, 102, id)
				require.Equal(t, marker+"existing report", body)
			} else {
				require.Zero(t, id)
				require.Empty(t, body)
			}
		})
	}
}
