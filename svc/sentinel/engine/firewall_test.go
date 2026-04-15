package engine

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	sentinelv1 "github.com/unkeyed/unkey/gen/proto/sentinel/v1"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

func TestFirewallExecutor_Deny(t *testing.T) {
	t.Parallel()

	e := &FirewallExecutor{}
	req := httptest.NewRequest("GET", "/xxx", nil)

	//nolint:exhaustruct
	cfg := &sentinelv1.Firewall{Action: sentinelv1.Action_ACTION_DENY}

	action, err := e.Execute(context.Background(), nil, req, cfg)
	require.Error(t, err)
	require.Equal(t, sentinelv1.Action_ACTION_DENY, action)

	urn, ok := fault.GetCode(err)
	require.True(t, ok)
	require.Equal(t, codes.Sentinel.Firewall.Denied.URN(), urn)
	require.Equal(t, "Forbidden", fault.UserFacingMessage(err))
}

func TestFirewallExecutor_Unspecified(t *testing.T) {
	t.Parallel()

	e := &FirewallExecutor{}
	req := httptest.NewRequest("GET", "/xxx", nil)

	//nolint:exhaustruct
	cfg := &sentinelv1.Firewall{Action: sentinelv1.Action_ACTION_UNSPECIFIED}

	action, err := e.Execute(context.Background(), nil, req, cfg)
	require.NoError(t, err)
	require.Equal(t, sentinelv1.Action_ACTION_UNSPECIFIED, action)
}

func TestFirewallActionLabel(t *testing.T) {
	t.Parallel()
	require.Equal(t, "deny", firewallActionLabel(sentinelv1.Action_ACTION_DENY))
	require.Equal(t, "unspecified", firewallActionLabel(sentinelv1.Action_ACTION_UNSPECIFIED))
}

// Execute should not touch the incoming request or HTTP response — it's a
// pure config evaluator. Regression guard for future changes.
func TestFirewallExecutor_DoesNotTouchRequest(t *testing.T) {
	t.Parallel()

	e := &FirewallExecutor{}
	req := httptest.NewRequest("GET", "/xxx", nil)
	req.Header.Set("X-Original", "yes")
	_, _ = e.Execute(context.Background(), nil, req, &sentinelv1.Firewall{ //nolint:exhaustruct
		Action: sentinelv1.Action_ACTION_DENY,
	})
	require.Equal(t, "yes", req.Header.Get("X-Original"))
	require.Equal(t, http.MethodGet, req.Method)
}
