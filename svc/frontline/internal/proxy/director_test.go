package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/frontline/internal/meta"
)

func TestInstanceDirector_RemovesMetadata(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 20, 12, 0, 0, 0, time.UTC)
	clk := clock.NewTestClock(now)
	metadata, err := meta.New("test-frontline-meta-signing-key")
	require.NoError(t, err)
	//nolint:exhaustruct
	svc := &service{
		instanceID: "frontline_1",
		platform:   "aws",
		region:     "us-east-1",
		clock:      clk,
		metadata:   metadata,
	}
	req, sess := newDirectorSession(t)
	token, err := metadata.Marshal(&meta.Metadata{Hops: []meta.Hop{{Region: "aws::us-east-1"}}})
	require.NoError(t, err)
	req.Header.Set(HeaderFrontlineMeta, token)

	svc.makeInstanceDirector(sess, now)(req)

	require.Equal(t, "aws::us-east-1", req.Header.Get(HeaderRegion))
	require.Equal(t, "frontline_1", req.Header.Get(HeaderFrontlineID))
	require.Empty(t, req.Header.Get(HeaderFrontlineMeta))
}

func TestRegionDirector_SetsMetadata(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 20, 12, 0, 0, 0, time.UTC)
	clk := clock.NewTestClock(now)
	metadata, err := meta.New("test-frontline-meta-signing-key")
	require.NoError(t, err)
	//nolint:exhaustruct
	svc := &service{
		instanceID: "frontline_1",
		platform:   "aws",
		region:     "us-east-1",
		clock:      clk,
		metadata:   metadata,
	}
	req, sess := newDirectorSession(t)
	signedMetadata, err := metadata.Marshal(&meta.Metadata{Hops: []meta.Hop{{Region: "aws::us-east-1"}}})
	require.NoError(t, err)

	svc.makeRegionDirector(sess, now, signedMetadata)(req)

	require.Equal(t, signedMetadata, req.Header.Get(HeaderFrontlineMeta))
	require.Empty(t, req.Header.Get(HeaderFrontlineID))
	require.Empty(t, req.Header.Get(HeaderRegion))
	require.Empty(t, req.Header.Get(HeaderRequestID))
}

func newDirectorSession(t *testing.T) (*http.Request, *zen.Session) {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "https://example.com", nil)
	w := httptest.NewRecorder()
	//nolint:exhaustruct
	sess := &zen.Session{}
	require.NoError(t, sess.Init(w, req, 0))
	return req, sess
}
