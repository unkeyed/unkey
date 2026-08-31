package ratelimit

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	frontlinev1 "github.com/unkeyed/unkey/gen/proto/frontline/v1"
	"github.com/unkeyed/unkey/svc/frontline/internal/policies/principal"
)

func pathSource() *frontlinev1.RateLimitIdentifier {
	return &frontlinev1.RateLimitIdentifier{
		Source: &frontlinev1.RateLimitIdentifier_Path{Path: &frontlinev1.PathKey{}},
	}
}

func headerSource(name string) *frontlinev1.RateLimitIdentifier {
	return &frontlinev1.RateLimitIdentifier{
		Source: &frontlinev1.RateLimitIdentifier_Header{Header: &frontlinev1.HeaderKey{Name: name}},
	}
}

func subjectSource() *frontlinev1.RateLimitIdentifier {
	return &frontlinev1.RateLimitIdentifier{
		Source: &frontlinev1.RateLimitIdentifier_AuthenticatedSubject{
			AuthenticatedSubject: &frontlinev1.AuthenticatedSubjectKey{},
		},
	}
}

func newRequest(t *testing.T, path string, headers map[string]string) *http.Request {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, "http://example.com"+path, nil)
	require.NoError(t, err)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return req
}

func TestExtractIdentifier(t *testing.T) {
	t.Run("single identifier keeps its raw value", func(t *testing.T) {
		req := newRequest(t, "/v1/search", nil)
		cfg := &frontlinev1.RateLimit{Identifier: pathSource()}
		require.Equal(t, "/v1/search", extractIdentifier(nil, req, cfg, nil))
	})

	t.Run("compound identifiers join in order", func(t *testing.T) {
		req := newRequest(t, "/v1/search", nil)
		cfg := &frontlinev1.RateLimit{
			Identifiers: []*frontlinev1.RateLimitIdentifier{subjectSource(), pathSource()},
		}
		p := &principal.Principal{Subject: "key_123"}
		require.Equal(t, "key_123:/v1/search", extractIdentifier(nil, req, cfg, p))
	})

	t.Run("separator bytes in parts cannot collide across positions", func(t *testing.T) {
		// ("a:b", "c") and ("a", "b:c") must produce different keys.
		reqA := newRequest(t, "/p", map[string]string{"X-A": "a:b", "X-B": "c"})
		reqB := newRequest(t, "/p", map[string]string{"X-A": "a", "X-B": "b:c"})
		cfg := &frontlinev1.RateLimit{
			Identifiers: []*frontlinev1.RateLimitIdentifier{headerSource("X-A"), headerSource("X-B")},
		}
		keyA := extractIdentifier(nil, reqA, cfg, nil)
		keyB := extractIdentifier(nil, reqB, cfg, nil)
		require.NotEqual(t, keyA, keyB)
	})

	t.Run("missing part falls back to unknown bucket", func(t *testing.T) {
		req := newRequest(t, "/v1/search", nil)
		cfg := &frontlinev1.RateLimit{
			Identifiers: []*frontlinev1.RateLimitIdentifier{subjectSource(), pathSource()},
		}
		// No principal: subject is unresolvable but path still partitions.
		require.Equal(t, "unknown:/v1/search", extractIdentifier(nil, req, cfg, nil))
	})

	t.Run("all parts unresolvable yields empty key", func(t *testing.T) {
		req := newRequest(t, "/v1/search", nil)
		cfg := &frontlinev1.RateLimit{
			Identifiers: []*frontlinev1.RateLimitIdentifier{subjectSource(), headerSource("X-Missing")},
		}
		require.Equal(t, "", extractIdentifier(nil, req, cfg, nil))
	})

	t.Run("no sources configured yields empty key", func(t *testing.T) {
		req := newRequest(t, "/v1/search", nil)
		require.Equal(t, "", extractIdentifier(nil, req, &frontlinev1.RateLimit{}, nil))
	})

	t.Run("single unresolvable identifier yields empty key", func(t *testing.T) {
		req := newRequest(t, "/v1/search", nil)
		cfg := &frontlinev1.RateLimit{Identifier: subjectSource()}
		require.Equal(t, "", extractIdentifier(nil, req, cfg, nil))
	})
}

// BenchmarkExtractIdentifier runs on the per-request hot path, so key
// composition must stay cheap: one buffer, one pass per part.
func BenchmarkExtractIdentifier(b *testing.B) {
	req, err := http.NewRequest(http.MethodGet, "http://example.com/v1/search", nil)
	require.NoError(b, err)
	p := &principal.Principal{Subject: "key_1234567890"}

	b.Run("legacy single", func(b *testing.B) {
		cfg := &frontlinev1.RateLimit{Identifier: pathSource()}
		b.ReportAllocs()
		for b.Loop() {
			extractIdentifier(nil, req, cfg, nil)
		}
	})

	b.Run("compound clean", func(b *testing.B) {
		cfg := &frontlinev1.RateLimit{
			Identifiers: []*frontlinev1.RateLimitIdentifier{subjectSource(), pathSource()},
		}
		b.ReportAllocs()
		for b.Loop() {
			extractIdentifier(nil, req, cfg, p)
		}
	})

	b.Run("compound escaped", func(b *testing.B) {
		cfg := &frontlinev1.RateLimit{
			Identifiers: []*frontlinev1.RateLimitIdentifier{subjectSource(), pathSource()},
		}
		escaped := &principal.Principal{Subject: `key:with\separators`}
		b.ReportAllocs()
		for b.Loop() {
			extractIdentifier(nil, req, cfg, escaped)
		}
	})
}
