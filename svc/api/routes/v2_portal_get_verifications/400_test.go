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
// those keys being idle, so the client would render a wrong answer. The
// account-wide series is unaffected and still answers.
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
	}

	headers := h.CreatePortalSession(workspace.ID, externalA, []string{api.KeyAuthID.String}, []string{"analytics:read"})

	req := Request{
		StartTime: now - int64(time.Hour/time.Millisecond),
		EndTime:   now + int64(time.Minute/time.Millisecond),
		PerKey:    ptr.P(true),
	}

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		res := testutil.CallRoute[Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(c, 400, res.Status, "a breakout past the cap must be rejected, not truncated")
	}, 30*time.Second, time.Second)

	withoutBreakout := req
	withoutBreakout.PerKey = nil

	res := testutil.CallRoute[Request, Response](h, route, headers, withoutBreakout)
	require.Equal(t, 200, res.Status, "the account-wide series still answers past the per-key cap")
	require.Equal(t, int64(2), sumTotals(res.Body.Data))
	require.Nil(t, res.Body.Keys)
}
