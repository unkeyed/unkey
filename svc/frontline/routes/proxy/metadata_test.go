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
	"github.com/unkeyed/unkey/pkg/jwt"
	"github.com/unkeyed/unkey/svc/frontline/internal/meta"
	"github.com/unkeyed/unkey/svc/frontline/internal/proxy"
)

func TestRequestHops_WithoutMetadataStartsWithEmptyHistory(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "https://example.com", nil)
	hops, err := requestHops(req, nil)
	require.NoError(t, err)
	require.Empty(t, hops)
}

func TestRequestHops_VerifiesAndRemovesMetadata(t *testing.T) {
	t.Parallel()

	codec := newMetadataCodec(t, "shared-signing-key")
	payload := validMetadataPayload()
	req := requestWithMetadata(t, codec, payload)

	hops, err := requestHops(req, codec)
	require.NoError(t, err)
	require.Equal(t, payload.Hops, hops)
	require.Empty(t, req.Header.Values(proxy.HeaderFrontlineMeta))
}

func TestRequestHops_IgnoresMalformedMetadata(t *testing.T) {
	t.Parallel()

	codec := newMetadataCodec(t, "shared-signing-key")
	req := httptest.NewRequest(http.MethodGet, "https://example.com", nil)
	req.Header.Set(proxy.HeaderFrontlineMeta, "client-controlled-value")

	hops, err := requestHops(req, codec)
	require.NoError(t, err)
	require.Empty(t, hops)
	require.Empty(t, req.Header.Values(proxy.HeaderFrontlineMeta))
}

func TestRequestHops_IgnoresWrongSigningKey(t *testing.T) {
	t.Parallel()

	signer := newMetadataCodec(t, "signing-key-a")
	verifier := newMetadataCodec(t, "signing-key-b")
	req := requestWithMetadata(t, signer, validMetadataPayload())

	hops, err := requestHops(req, verifier)
	require.NoError(t, err)
	require.Empty(t, hops)
	require.Empty(t, req.Header.Values(proxy.HeaderFrontlineMeta))
}

func TestRequestHops_IgnoresMissingExpiry(t *testing.T) {
	t.Parallel()

	codec := newMetadataCodec(t, "shared-signing-key")
	payload := validMetadataPayload()
	payload.ExpiresAt = 0
	req := requestWithMetadata(t, codec, payload)

	hops, err := requestHops(req, codec)
	require.NoError(t, err)
	require.Empty(t, hops)
	require.Empty(t, req.Header.Values(proxy.HeaderFrontlineMeta))
}

func TestRequestHops_IgnoresExpiredMetadata(t *testing.T) {
	t.Parallel()

	codec := newMetadataCodec(t, "shared-signing-key")
	payload := validMetadataPayload()
	payload.ExpiresAt = time.Now().Add(-time.Minute).Unix()
	req := requestWithMetadata(t, codec, payload)

	hops, err := requestHops(req, codec)
	require.NoError(t, err)
	require.Empty(t, hops)
	require.Empty(t, req.Header.Values(proxy.HeaderFrontlineMeta))
}

func TestRequestHops_IgnoresDuplicateHeaders(t *testing.T) {
	t.Parallel()

	codec := newMetadataCodec(t, "shared-signing-key")
	req := requestWithMetadata(t, codec, validMetadataPayload())
	req.Header.Add(proxy.HeaderFrontlineMeta, req.Header.Get(proxy.HeaderFrontlineMeta))

	hops, err := requestHops(req, codec)
	require.NoError(t, err)
	require.Empty(t, hops)
	require.Empty(t, req.Header.Values(proxy.HeaderFrontlineMeta))
}

func TestRequestHops_IgnoresEmptyHeader(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "https://example.com", nil)
	req.Header.Set(proxy.HeaderFrontlineMeta, "")

	hops, err := requestHops(req, nil)
	require.NoError(t, err)
	require.Empty(t, hops)
	require.Empty(t, req.Header.Values(proxy.HeaderFrontlineMeta))
}

func TestRequestHops_IgnoresOversizedHeader(t *testing.T) {
	t.Parallel()

	codec := newMetadataCodec(t, "shared-signing-key")
	req := httptest.NewRequest(http.MethodGet, "https://example.com", nil)
	req.Header.Set(proxy.HeaderFrontlineMeta, strings.Repeat("a", 4097))

	hops, err := requestHops(req, codec)
	require.NoError(t, err)
	require.Empty(t, hops)
	require.Empty(t, req.Header.Values(proxy.HeaderFrontlineMeta))
}

func TestRequestHops_RejectsMetadataWithoutCodec(t *testing.T) {
	t.Parallel()

	codec := newMetadataCodec(t, "shared-signing-key")
	req := requestWithMetadata(t, codec, validMetadataPayload())

	_, err := requestHops(req, nil)
	requireFrontlineMetadataError(t, err)
	require.Empty(t, req.Header.Values(proxy.HeaderFrontlineMeta))
}

func newMetadataCodec(t *testing.T, signingKey string) *meta.Codec {
	t.Helper()

	codec, err := meta.New(signingKey)
	require.NoError(t, err)
	return codec
}

func requestWithMetadata(t *testing.T, codec *meta.Codec, payload *meta.Metadata) *http.Request {
	t.Helper()

	token, err := codec.Marshal(payload)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodGet, "https://example.com", nil)
	req.Header.Set(proxy.HeaderFrontlineMeta, token)
	return req
}

func validMetadataPayload() *meta.Metadata {
	return &meta.Metadata{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: time.Now().Add(time.Hour).Unix(),
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

func requireFrontlineMetadataError(t *testing.T, err error) {
	t.Helper()

	require.Error(t, err)
	code, ok := fault.GetCode(err)
	require.True(t, ok)
	require.Equal(t, codes.Frontline.Internal.InternalServerError.URN(), code)
}
