package github

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/circuitbreaker"
)

func TestNoopPublicCallsShareResilientClient(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch requests.Add(1) {
		case 1:
			w.WriteHeader(http.StatusServiceUnavailable)
		case 2:
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	t.Cleanup(server.Close)
	client := NewNoop()
	client.baseURL = server.URL
	t.Cleanup(client.httpClient.CloseIdleConnections)
	public, err := client.IsRepoPublic("acme/app")
	require.NoError(t, err)
	require.True(t, public)
	for range 5 {
		_, err := client.IsRepoPublic("acme/app")
		require.Error(t, err)
	}
	_, err = client.GetBranchHeadCommitPublic("acme/app", "main")
	require.ErrorIs(t, err, circuitbreaker.ErrTripped)
}

func TestClientIsRepoPublicRetriesUnavailableEndpoint(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	client, err := NewClient(ClientConfig{
		AppID: 0,
		PrivateKeyPEM: string(pem.EncodeToMemory(&pem.Block{
			Type: "RSA PRIVATE KEY", Headers: nil, Bytes: x509.MarshalPKCS1PrivateKey(key),
		})),
		WebhookSecret: "",
	})
	require.NoError(t, err)
	t.Cleanup(client.httpClient.CloseIdleConnections)
	t.Cleanup(client.tokenCache.Close)
	t.Cleanup(client.scopedTokenCache.Close)
	t.Cleanup(client.collaboratorCache.Close)
	t.Cleanup(client.visibilityCache.Close)

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)
	client.baseURL = server.URL

	public, err := client.IsRepoPublic("acme/app")
	require.NoError(t, err)
	require.True(t, public)
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
