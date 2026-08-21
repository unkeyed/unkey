package proxy

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/frontline/internal/meta"
)

func TestNew_RequiresMetadata(t *testing.T) {
	t.Parallel()

	//nolint:exhaustruct
	svc, err := New(Config{})
	require.ErrorContains(t, err, "metadata codec is required")
	require.Nil(t, svc)
}

func TestNew_RejectsNegativeMaxHops(t *testing.T) {
	t.Parallel()

	metadata, err := meta.New("test-frontline-meta-signing-key")
	require.NoError(t, err)

	//nolint:exhaustruct
	svc, err := New(Config{
		MaxHops:  -1,
		Metadata: metadata,
	})
	require.ErrorContains(t, err, "max hops must not be negative")
	require.Nil(t, svc)
}

func TestForwardToRegion_RejectsHopLimit(t *testing.T) {
	t.Parallel()

	clk := clock.New()
	metadata, err := meta.New("test-frontline-meta-signing-key")
	require.NoError(t, err)
	//nolint:exhaustruct
	svc, err := New(Config{
		InstanceID: "frontline_1",
		Platform:   "aws",
		Region:     "us-east-1",
		ApexDomain: "example.com",
		Clock:      clk,
		MaxHops:    3,
		Metadata:   metadata,
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "https://api.example.com", nil)
	w := httptest.NewRecorder()
	//nolint:exhaustruct
	sess := &zen.Session{}
	require.NoError(t, sess.Init(w, req, 0))

	err = svc.ForwardToRegion(context.Background(), sess, "us-west-2.aws", make([]meta.Hop, 3))
	require.Error(t, err)
	urn, ok := fault.GetCode(err)
	require.True(t, ok)
	require.Equal(t, codes.Frontline.Internal.InternalServerError.URN(), urn)
}

func TestForwardToRegion_AllowsLastHop(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Millisecond)
	clk := clock.NewTestClock(now)
	metadata, err := meta.New("test-frontline-meta-signing-key")
	require.NoError(t, err)
	transport, recorder := newMetadataTransport(t, metadata)
	//nolint:exhaustruct
	svc := &service{
		instanceID: "frontline_3",
		platform:   "aws",
		region:     "eu-west-1",
		apexDomain: "example.com",
		clock:      clk,
		transport:  transport,
		maxHops:    3,
		metadata:   metadata,
	}
	hops := []meta.Hop{
		{Region: "aws::us-east-1"},
		{Region: "aws::us-west-2"},
	}
	sess := newProxySession(t, httptest.NewRequest(http.MethodGet, "https://api.example.com", nil))

	err = svc.ForwardToRegion(context.Background(), sess, "eu-central-1.aws", hops)
	require.NoError(t, err)
	require.Len(t, recorder.seen, 1)
	require.Len(t, recorder.seen[0].Hops, 3)
}

func TestForwardToRegion_AppendsMetadataAtEachHop(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Millisecond)
	clk := clock.NewTestClock(now)
	metadata, err := meta.New("test-frontline-meta-signing-key")
	require.NoError(t, err)
	transport, recorder := newMetadataTransport(t, metadata)

	//nolint:exhaustruct
	first := &service{
		instanceID: "frontline_1",
		platform:   "aws",
		region:     "us-east-1",
		apexDomain: "example.com",
		clock:      clk,
		transport:  transport,
		maxHops:    3,
		metadata:   metadata,
	}
	req := httptest.NewRequest(http.MethodGet, "https://api.example.com/v1/items?page=2", nil)
	firstSession := newProxySession(t, req)
	ctx := WithRequestStartTime(context.Background(), now)
	require.NoError(t, first.ForwardToRegion(ctx, firstSession, "us-west-2.aws", nil))
	require.Len(t, recorder.seen, 1)
	require.Equal(t, now.Add(frontlineMetadataTTL).Unix(), recorder.seen[0].ExpiresAt)
	require.Equal(t, []meta.Hop{
		{
			Region:        "aws::us-east-1",
			RequestID:     firstSession.RequestID(),
			FrontlineID:   "frontline_1",
			TimeUnixMilli: now.UnixMilli(),
		},
	}, recorder.seen[0].Hops)
	require.Empty(t, req.Header.Get(HeaderFrontlineMeta))

	incomingHops := recorder.seen[0].Hops
	secondNow := clk.Tick(250 * time.Millisecond)
	//nolint:exhaustruct
	second := &service{
		instanceID: "frontline_2",
		platform:   "aws",
		region:     "us-west-2",
		apexDomain: "example.com",
		clock:      clk,
		transport:  transport,
		maxHops:    3,
		metadata:   metadata,
	}
	secondReq := httptest.NewRequest(http.MethodGet, "https://api.example.com/v1/items?page=2", nil)
	secondSession := newProxySession(t, secondReq)
	ctx = WithRequestStartTime(context.Background(), secondNow)
	require.NoError(t, second.ForwardToRegion(ctx, secondSession, "eu-west-1.aws", incomingHops))
	require.Len(t, recorder.seen, 2)
	require.Equal(t, secondNow.Add(frontlineMetadataTTL).Unix(), recorder.seen[1].ExpiresAt)
	require.Equal(t, []meta.Hop{
		{
			Region:        "aws::us-east-1",
			RequestID:     firstSession.RequestID(),
			FrontlineID:   "frontline_1",
			TimeUnixMilli: now.UnixMilli(),
		},
		{
			Region:        "aws::us-west-2",
			RequestID:     secondSession.RequestID(),
			FrontlineID:   "frontline_2",
			TimeUnixMilli: secondNow.UnixMilli(),
		},
	}, recorder.seen[1].Hops)
	require.Empty(t, secondReq.Header.Get(HeaderFrontlineMeta))
	require.NotEqual(t, recorder.tokens[0], recorder.tokens[1])
}

func TestForwardToRegion_PreservesDuplicateRegions(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Millisecond)
	clk := clock.NewTestClock(now)
	metadata, err := meta.New("test-frontline-meta-signing-key")
	require.NoError(t, err)
	transport, recorder := newMetadataTransport(t, metadata)
	//nolint:exhaustruct
	svc := &service{
		instanceID: "frontline_2",
		platform:   "aws",
		region:     "us-east-1",
		apexDomain: "example.com",
		clock:      clk,
		transport:  transport,
		maxHops:    3,
		metadata:   metadata,
	}
	hops := []meta.Hop{{Region: "aws::us-east-1"}}
	sess := newProxySession(t, httptest.NewRequest(http.MethodGet, "https://api.example.com", nil))

	err = svc.ForwardToRegion(context.Background(), sess, "us-west-2.aws", hops)
	require.NoError(t, err)
	require.Len(t, recorder.seen, 1)
	require.Equal(t, "aws::us-east-1", recorder.seen[0].Hops[0].Region)
	require.Equal(t, "aws::us-east-1", recorder.seen[0].Hops[1].Region)
}

type metadataTransport struct {
	codec  *meta.Codec
	seen   []*meta.Metadata
	tokens []string
}

func newMetadataTransport(t *testing.T, codec *meta.Codec) (*http.Transport, *metadataTransport) {
	t.Helper()

	recorder := &metadataTransport{codec: codec}
	transport := &http.Transport{}
	transport.RegisterProtocol("https", recorder)
	t.Cleanup(transport.CloseIdleConnections)
	return transport, recorder
}

func (t *metadataTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	token := req.Header.Get(HeaderFrontlineMeta)
	metadata, err := t.codec.Unmarshal(token)
	if err != nil {
		return nil, err
	}
	t.seen = append(t.seen, metadata)
	t.tokens = append(t.tokens, token)
	//nolint:exhaustruct
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader("ok")),
		Request:    req,
	}, nil
}

func newProxySession(t *testing.T, req *http.Request) *zen.Session {
	t.Helper()
	//nolint:exhaustruct
	sess := &zen.Session{}
	require.NoError(t, sess.Init(httptest.NewRecorder(), req, 0))
	return sess
}
