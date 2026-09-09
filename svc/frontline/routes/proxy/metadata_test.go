package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/paseto"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/frontline/internal/meta"
	"github.com/unkeyed/unkey/svc/frontline/internal/proxy"
)

const (
	metadataSigningKey          = "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"
	alternateMetadataSigningKey = "202122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f"
)

func TestRequestHops_WithoutMetadataStartsWithEmptyHistory(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "https://example.com", nil)
	hops, err := requestHops(newMetadataSession(t, req), nil, metadataRequestTime())
	require.NoError(t, err)
	require.Empty(t, hops)
}

func TestRequestHops_VerifiesAndRemovesMetadata(t *testing.T) {
	t.Parallel()

	codec := newMetadataCodec(t, metadataSigningKey)
	metadata := validMetadata()
	req := requestWithMetadata(t, codec, metadata)

	hops, err := requestHops(newMetadataSession(t, req), codec, metadataRequestTime())
	require.NoError(t, err)
	require.Equal(t, metadata.Hops, hops)
	require.Empty(t, req.Header.Values(proxy.HeaderFrontlineMeta))
}

func TestRequestHops_IgnoresMalformedMetadata(t *testing.T) {
	t.Parallel()

	codec := newMetadataCodec(t, metadataSigningKey)
	req := httptest.NewRequest(http.MethodGet, "https://example.com", nil)
	req.Header.Set(proxy.HeaderFrontlineMeta, "client-controlled-value")

	hops, err := requestHops(newMetadataSession(t, req), codec, metadataRequestTime())
	require.NoError(t, err)
	require.Empty(t, hops)
	require.Empty(t, req.Header.Values(proxy.HeaderFrontlineMeta))
}

func TestRequestHops_IgnoresWrongSigningKey(t *testing.T) {
	t.Parallel()

	signer := newMetadataCodec(t, metadataSigningKey)
	verifier := newMetadataCodec(t, alternateMetadataSigningKey)
	req := requestWithMetadata(t, signer, validMetadata())

	hops, err := requestHops(newMetadataSession(t, req), verifier, metadataRequestTime())
	require.NoError(t, err)
	require.Empty(t, hops)
	require.Empty(t, req.Header.Values(proxy.HeaderFrontlineMeta))
}

func TestRequestHops_IgnoresMissingExpiry(t *testing.T) {
	t.Parallel()

	codec := newMetadataCodec(t, metadataSigningKey)
	metadata := validMetadata()
	metadata.ExpiresAt = time.Time{}
	req := requestWithMetadata(t, codec, metadata)

	hops, err := requestHops(newMetadataSession(t, req), codec, metadataRequestTime())
	require.NoError(t, err)
	require.Empty(t, hops)
	require.Empty(t, req.Header.Values(proxy.HeaderFrontlineMeta))
}

func TestRequestHops_IgnoresExpiredMetadata(t *testing.T) {
	t.Parallel()

	codec := newMetadataCodec(t, metadataSigningKey)
	metadata := validMetadata()
	metadata.ExpiresAt = metadataRequestTime().Add(-time.Minute)
	req := requestWithMetadata(t, codec, metadata)

	hops, err := requestHops(newMetadataSession(t, req), codec, metadataRequestTime())
	require.NoError(t, err)
	require.Empty(t, hops)
	require.Empty(t, req.Header.Values(proxy.HeaderFrontlineMeta))
}

func TestRequestHops_IgnoresMetadataAtExpiry(t *testing.T) {
	t.Parallel()

	codec := newMetadataCodec(t, metadataSigningKey)
	metadata := validMetadata()
	metadata.ExpiresAt = metadataRequestTime()
	req := requestWithMetadata(t, codec, metadata)

	hops, err := requestHops(newMetadataSession(t, req), codec, metadataRequestTime())
	require.NoError(t, err)
	require.Empty(t, hops)
	require.Empty(t, req.Header.Values(proxy.HeaderFrontlineMeta))
}

func TestRequestHops_IgnoresDuplicateHeaders(t *testing.T) {
	t.Parallel()

	codec := newMetadataCodec(t, metadataSigningKey)
	req := requestWithMetadata(t, codec, validMetadata())
	req.Header.Add(proxy.HeaderFrontlineMeta, req.Header.Get(proxy.HeaderFrontlineMeta))

	hops, err := requestHops(newMetadataSession(t, req), codec, metadataRequestTime())
	require.NoError(t, err)
	require.Empty(t, hops)
	require.Empty(t, req.Header.Values(proxy.HeaderFrontlineMeta))
}

func TestRequestHops_IgnoresEmptyHeader(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "https://example.com", nil)
	req.Header.Set(proxy.HeaderFrontlineMeta, "")

	hops, err := requestHops(newMetadataSession(t, req), nil, metadataRequestTime())
	require.NoError(t, err)
	require.Empty(t, hops)
	require.Empty(t, req.Header.Values(proxy.HeaderFrontlineMeta))
}

func TestRequestHops_IgnoresOversizedHeader(t *testing.T) {
	t.Parallel()

	codec := newMetadataCodec(t, metadataSigningKey)
	req := httptest.NewRequest(http.MethodGet, "https://example.com", nil)
	req.Header.Set(proxy.HeaderFrontlineMeta, strings.Repeat("a", 4097))

	hops, err := requestHops(newMetadataSession(t, req), codec, metadataRequestTime())
	require.NoError(t, err)
	require.Empty(t, hops)
	require.Empty(t, req.Header.Values(proxy.HeaderFrontlineMeta))
}

func TestRequestHops_RejectsMetadataWithoutCodec(t *testing.T) {
	t.Parallel()

	codec := newMetadataCodec(t, metadataSigningKey)
	req := requestWithMetadata(t, codec, validMetadata())

	_, err := requestHops(newMetadataSession(t, req), nil, metadataRequestTime())
	requireFrontlineMetadataError(t, err)
	require.Empty(t, req.Header.Values(proxy.HeaderFrontlineMeta))
}

func TestRequestHops_ClientIP(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name    string
		ip      string
		key     string
		expires time.Time
		want    string
	}{
		{"ipv4", "198.51.100.42", metadataSigningKey, metadataRequestTime().Add(time.Minute), "198.51.100.42"},
		{"ipv6", "2001:db8::42", metadataSigningKey, metadataRequestTime().Add(time.Minute), "2001:db8::42"},
		{"old peer", "", metadataSigningKey, metadataRequestTime().Add(time.Minute), "10.1.2.3"},
		{"invalid ip", "attacker", metadataSigningKey, metadataRequestTime().Add(time.Minute), "10.1.2.3"},
		{"zone", "fe80::1%eth0", metadataSigningKey, metadataRequestTime().Add(time.Minute), "10.1.2.3"},
		{"wrong key", "198.51.100.42", alternateMetadataSigningKey, metadataRequestTime().Add(time.Minute), "10.1.2.3"},
		{"expired", "198.51.100.42", metadataSigningKey, metadataRequestTime(), "10.1.2.3"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			metadata := validMetadata()
			metadata.ClientIP = tt.ip
			metadata.ExpiresAt = tt.expires
			req := requestWithMetadata(t, newMetadataCodec(t, tt.key), metadata)
			req.RemoteAddr = "10.1.2.3:443"
			req.Header.Set("X-Forwarded-For", "203.0.113.77")
			sess := newMetadataSession(t, req)
			_, err := requestHops(sess, newMetadataCodec(t, metadataSigningKey), metadataRequestTime())
			require.NoError(t, err)
			require.Equal(t, tt.want, sess.Location())
			require.Equal(t, "10.1.2.3:443", req.RemoteAddr)
			require.Empty(t, req.Header.Get(proxy.HeaderFrontlineMeta))
		})
	}
}

func newMetadataCodec(t *testing.T, signingKey string) *meta.Codec {
	t.Helper()

	codec, err := meta.New(signingKey)
	require.NoError(t, err)
	return codec
}

func newMetadataSession(t *testing.T, req *http.Request) *zen.Session {
	t.Helper()
	sess := &zen.Session{}
	require.NoError(t, sess.Init(httptest.NewRecorder(), req, 0))
	return sess
}

func requestWithMetadata(t *testing.T, codec *meta.Codec, metadata *meta.Metadata) *http.Request {
	t.Helper()

	token, err := codec.Marshal(metadata)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodGet, "https://example.com", nil)
	req.Header.Set(proxy.HeaderFrontlineMeta, token)
	return req
}

func validMetadata() *meta.Metadata {
	return &meta.Metadata{
		Claims: paseto.Claims{
			ExpiresAt: metadataRequestTime().Add(time.Hour),
		},
		Hops: []meta.Hop{
			{
				Region:        "aws::us-east-1",
				RequestID:     "req_123",
				FrontlineID:   "frontline_123",
				TimeUnixMilli: 1_777_000_000_123,
			},
		},
	}
}

func metadataRequestTime() time.Time {
	return time.Date(2026, time.August, 26, 12, 0, 0, 0, time.UTC)
}

func requireFrontlineMetadataError(t *testing.T, err error) {
	t.Helper()

	require.Error(t, err)
	code, ok := fault.GetCode(err)
	require.True(t, ok)
	require.Equal(t, codes.Frontline.Internal.InternalServerError.URN(), code)
}
