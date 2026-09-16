package handler_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_get_verifications"
)

// TestPortalSessionAnalyticsRejectsOversizedWindow verifies the query window is
// bounded by the workspace's log retention. This endpoint runs on the shared
// ClickHouse connection, so an unbounded window would let an end user force an
// arbitrarily large scan.
func TestPortalSessionAnalyticsRejectsOversizedWindow(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})

	workspace := h.CreateWorkspace()
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})
	// SetupAnalytics defaults LogsRetentionDays to 30.
	h.SetupAnalytics(workspace.ID)

	route := newHandler(h)
	h.Register(route, h.PortalMiddleware()...)

	headers := h.CreatePortalSession(workspace.ID, "portal_user_A", []string{api.KeyAuthID.String}, []string{"analytics:read"})

	now := time.Now().UnixMilli()
	dayMs := int64(24 * time.Hour / time.Millisecond)

	// 40-day window against a 30-day retention must be rejected.
	req := Request{
		StartTime: now - 40*dayMs,
		EndTime:   now,
	}

	res := testutil.CallRoute[Request, openapi.BadRequestErrorResponse](h, route, headers, req)
	require.Equal(t, 400, res.Status, "window wider than retention must be rejected")

	// The retention ceiling carries its own code, so a client can branch on it
	// without matching the human message. It is the same code the operator
	// analytics path raises against the same limit.
	require.NotNil(t, res.Body)
	require.Equal(t, codes.User.BadRequest.QueryRangeExceedsRetention.DocsURL(), res.Body.Error.Type,
		"retention rejection must be distinguishable from ordinary input validation")

	// An inverted window is ordinary input validation, not a retention ceiling,
	// so the two stay distinguishable.
	inverted := testutil.CallRoute[Request, openapi.BadRequestErrorResponse](h, route, headers, Request{
		StartTime: now,
		EndTime:   now - dayMs,
	})
	require.Equal(t, 400, inverted.Status)
	require.NotNil(t, inverted.Body)
	require.Equal(t, codes.App.Validation.InvalidInput.DocsURL(), inverted.Body.Error.Type)

	// A window within retention is accepted.
	ok := Request{
		StartTime: now - 10*dayMs,
		EndTime:   now,
	}
	okRes := testutil.CallRoute[Request, Response](h, route, headers, ok)
	require.Equal(t, 200, okRes.Status, "window within retention must be accepted")
}

// TestPortalSessionAnalyticsRejectsOverflowingWindow pins that the retention
// ceiling cannot be stepped over by making the window arithmetic overflow. A
// large negative start with a large positive end wraps endTime-startTime
// negative, which reads as "smaller than retention" and would otherwise select
// minute granularity over an unbounded range on the shared connection.
func TestPortalSessionAnalyticsRejectsOverflowingWindow(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})

	workspace := h.CreateWorkspace()
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})
	h.SetupAnalytics(workspace.ID)

	route := newHandler(h)
	h.Register(route, h.PortalMiddleware()...)

	headers := h.CreatePortalSession(workspace.ID, "portal_user_A", []string{api.KeyAuthID.String}, []string{"analytics:read"})

	res := testutil.CallRoute[Request, openapi.BadRequestErrorResponse](h, route, headers, Request{
		StartTime: -8_000_000_000_000_000_000,
		EndTime:   8_000_000_000_000_000_000,
	})
	require.Equal(t, 400, res.Status, "an overflowing window must be rejected, not served")
	require.NotNil(t, res.Body)
	require.Equal(t, codes.App.Validation.InvalidInput.DocsURL(), res.Body.Error.Type)
}

// TestPortalSessionAnalyticsRejectsOversizedPerKeyBreakout verifies the per-key
// breakout is rejected rather than truncated once the session has more keys with
// traffic than the cap allows. A short array would be indistinguishable from
// those keys being idle, so the client would render a wrong answer.
func TestPortalSessionAnalyticsRejectsOversizedPerKeyBreakout(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})

	workspace := h.CreateWorkspace()
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})
	h.SetupAnalytics(workspace.ID)

	route := newHandlerWithKeyCap(h, 1)
	h.Register(route, h.PortalMiddleware()...)

	externalA := "portal_user_A"
	identityA := h.CreateIdentity(seed.CreateIdentityRequest{
		WorkspaceID: workspace.ID,
		ExternalID:  externalA,
	})

	now := time.Now().UnixMilli()
	var lastKey seed.CreateKeyResponse
	for range 2 {
		key := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			IdentityID:  ptr.P(identityA.ID),
		})
		h.KeyVerifications.Buffer(schema.KeyVerification{
			RequestID:   uid.New(uid.RequestPrefix),
			Time:        now,
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			KeyID:       key.KeyID,
			Region:      "us-west-1",
			Outcome:     "VALID",
			IdentityID:  identityA.ID,
			ExternalID:  externalA,
			Tags:        []string{},
		})
		lastKey = key
	}

	headers := h.CreatePortalSession(workspace.ID, externalA, []string{api.KeyAuthID.String}, []string{"analytics:read"})

	req := Request{
		StartTime: now - int64(time.Hour/time.Millisecond),
		EndTime:   now + int64(time.Minute/time.Millisecond),
	}

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		res := testutil.CallRoute[Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(c, 400, res.Status, "a breakout past the cap must be rejected, not truncated")
		require.NotNil(c, res.Body)

		// The route has three ways to answer 400, so asserting the status alone
		// would keep passing if the cap stopped firing and something else
		// rejected the request instead.
		require.Equal(c, codes.User.BadRequest.PerKeyBreakoutTooLarge.DocsURL(), res.Body.Error.Type,
			"the cap must be distinguishable from ordinary input validation")
	}, 30*time.Second, time.Second)

	narrowed := req
	narrowed.KeyId = ptr.P(lastKey.KeyID)

	res := testutil.CallRoute[Request, Response](h, route, headers, narrowed)
	require.Equal(t, 200, res.Status, "a single named key stays within the cap")
	require.Equal(t, int64(1), sumTotals(res.Body.Keys))
}

// TestPortalSessionAnalyticsRejectsOversizedResponse pins the response-size
// ceiling. The key cap bounds how many series a breakout carries, not how large
// they are, so a body past the shared analytics limit is refused rather than
// serialized off the shared connection.
func TestPortalSessionAnalyticsRejectsOversizedResponse(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})

	workspace := h.CreateWorkspace()
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})
	h.SetupAnalytics(workspace.ID)

	route := newHandlerWithLimits(h, handler.DefaultMaxPerKeySeries, 4096)
	h.Register(route, h.PortalMiddleware()...)

	externalA := "portal_user_A"
	identityA := h.CreateIdentity(seed.CreateIdentityRequest{
		WorkspaceID: workspace.ID,
		ExternalID:  externalA,
	})

	// A minute-granularity window over several keys is what makes the body grow:
	// every key carries a bucket per minute it saw traffic. The ceiling is
	// lowered rather than seeding a production-scale body.
	now := time.Now().UnixMilli()
	minuteMs := int64(time.Minute / time.Millisecond)
	var lastKey seed.CreateKeyResponse
	for range 8 {
		key := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			IdentityID:  ptr.P(identityA.ID),
		})
		for i := range 10 {
			h.KeyVerifications.Buffer(schema.KeyVerification{
				RequestID:   uid.New(uid.RequestPrefix),
				Time:        now - int64(i)*minuteMs,
				WorkspaceID: workspace.ID,
				KeySpaceID:  api.KeyAuthID.String,
				KeyID:       key.KeyID,
				Region:      "us-west-1",
				Outcome:     "VALID",
				IdentityID:  identityA.ID,
				ExternalID:  externalA,
				Tags:        []string{},
			})
		}
		lastKey = key
	}

	headers := h.CreatePortalSession(workspace.ID, externalA, []string{api.KeyAuthID.String}, []string{"analytics:read"})

	// Every key carries a bucket per minute of the window, so eight keys cross
	// the ceiling a single key stays under.
	req := Request{
		StartTime: now - 10*minuteMs,
		EndTime:   now + minuteMs,
	}

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		res := testutil.CallRoute[Request, openapi.UnprocessableEntityErrorResponse](h, route, headers, req)
		require.Equal(c, 422, res.Status, "a body past the analytics size ceiling must be refused")
	}, 60*time.Second, time.Second)

	narrowed := req
	narrowed.KeyId = ptr.P(lastKey.KeyID)
	ok := testutil.CallRoute[Request, Response](h, route, headers, narrowed)
	require.Equal(t, 200, ok.Status, "a single key stays under the size ceiling")
}
