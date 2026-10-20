package git

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os/exec"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetInfoRetriesAuthorLookupAndKeepsGitMetadataOnFailure(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, args := range [][]string{
		{"init", "-b", "main"},
		{"-c", "user.name=Test", "-c", "user.email=test@example.com", "-c", "commit.gpgsign=false", "-c", "core.hooksPath=/dev/null", "commit", "--allow-empty", "-m", "test commit"},
		{"remote", "add", "origin", "https://github.com/acme/app.git"},
	} {
		output, err := exec.Command("git", args...).CombinedOutput()
		require.NoError(t, err, "%s", output)
	}

	var requests atomic.Int32
	var fail atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if fail.Load() || r.Header.Get("User-Agent") != "unkey-cli" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		if requests.Add(1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		var response githubCommitResponse
		response.Author.Login = "test-author"
		response.Author.AvatarURL = "https://example.com/avatar.png"
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	target, err := url.Parse(server.URL)
	require.NoError(t, err)
	original := githubHTTPClient.Transport
	next := original
	if next == nil {
		next = http.DefaultTransport
	}
	githubHTTPClient.Transport = redirectTransport{target: target, next: next}
	t.Cleanup(func() {
		githubHTTPClient.Transport = original
		githubHTTPClient.CloseIdleConnections()
	})

	info := GetInfo()
	require.Equal(t, "test-author", info.AuthorHandle)
	require.Equal(t, "https://example.com/avatar.png", info.AuthorAvatarURL)
	fail.Store(true)
	failed := GetInfo()
	require.True(t, failed.IsRepo)
	require.Equal(t, info.CommitSHA, failed.CommitSHA)
	require.Equal(t, "test commit", failed.Message)
	require.Empty(t, failed.AuthorHandle)
	require.Empty(t, failed.AuthorAvatarURL)
}

type redirectTransport struct {
	target *url.URL
	next   http.RoundTripper
}

func (t redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.URL.Scheme = t.target.Scheme
	req.URL.Host = t.target.Host
	return t.next.RoundTrip(req)
}
