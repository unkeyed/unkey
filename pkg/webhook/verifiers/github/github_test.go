package github

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// sign produces an X-Hub-Signature-256 header for body: sha256=<hmac-sha256
// of the raw body>, the scheme GitHub documents for webhook deliveries.
func sign(secret, body string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(body))
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestVerifier(t *testing.T) {
	const secret = "gh_test_secret"
	body := `{"ref":"refs/heads/main","after":"abc123"}`

	request := func(mutate func(*http.Request)) *http.Request {
		r := httptest.NewRequest(http.MethodPost, "/webhooks/github", strings.NewReader(body))
		r.Header.Set("X-Hub-Signature-256", sign(secret, body))
		r.Header.Set("X-GitHub-Event", "push")
		r.Header.Set("X-GitHub-Delivery", "delivery-123")
		if mutate != nil {
			mutate(r)
		}
		return r
	}

	t.Run("valid signature yields event from headers", func(t *testing.T) {
		event, err := New(secret).Verify(request(nil))
		require.NoError(t, err)
		require.Equal(t, "delivery-123", event.ID)
		require.Equal(t, "push", event.Type)
		require.Equal(t, body, string(event.Payload))
	})

	t.Run("wrong secret is rejected", func(t *testing.T) {
		_, err := New("other_secret").Verify(request(nil))
		require.Error(t, err)
	})

	t.Run("missing signature header is rejected", func(t *testing.T) {
		_, err := New(secret).Verify(request(func(r *http.Request) {
			r.Header.Del("X-Hub-Signature-256")
		}))
		require.Error(t, err)
	})

	t.Run("tampered body is rejected", func(t *testing.T) {
		// Body differs from what the signature was computed over.
		r := request(func(r *http.Request) {
			r.Body = io.NopCloser(strings.NewReader(`{"ref":"refs/heads/evil"}`))
		})
		_, err := New(secret).Verify(r)
		require.Error(t, err)
	})

	t.Run("missing event header is rejected", func(t *testing.T) {
		_, err := New(secret).Verify(request(func(r *http.Request) {
			r.Header.Del("X-GitHub-Event")
		}))
		require.Error(t, err)
	})

	t.Run("missing delivery header is rejected", func(t *testing.T) {
		_, err := New(secret).Verify(request(func(r *http.Request) {
			r.Header.Del("X-GitHub-Delivery")
		}))
		require.Error(t, err)
	})
}
