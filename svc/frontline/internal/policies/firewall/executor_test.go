package firewall

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	frontlinev1 "github.com/unkeyed/unkey/gen/proto/frontline/v1"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

func TestFirewallExecutor_Deny(t *testing.T) {
	t.Parallel()

	e := &Executor{}
	req := httptest.NewRequest("GET", "/xxx", nil)

	//nolint:exhaustruct
	cfg := &frontlinev1.Firewall{Action: frontlinev1.Action_ACTION_DENY}

	action, err := e.Execute(context.Background(), nil, req, cfg)
	require.Error(t, err)
	require.Equal(t, frontlinev1.Action_ACTION_DENY, action)

	urn, ok := fault.GetCode(err)
	require.True(t, ok)
	require.Equal(t, codes.Frontline.Firewall.Denied.URN(), urn)
	require.Equal(t, "Forbidden", fault.UserFacingMessage(err))
}

func TestFirewallExecutor_Unspecified(t *testing.T) {
	t.Parallel()

	e := &Executor{}
	req := httptest.NewRequest("GET", "/xxx", nil)

	//nolint:exhaustruct
	cfg := &frontlinev1.Firewall{Action: frontlinev1.Action_ACTION_UNSPECIFIED}

	action, err := e.Execute(context.Background(), nil, req, cfg)
	require.NoError(t, err)
	require.Equal(t, frontlinev1.Action_ACTION_UNSPECIFIED, action)
}

func TestFirewallActionLabel(t *testing.T) {
	t.Parallel()
	require.Equal(t, "deny", ActionLabel(frontlinev1.Action_ACTION_DENY))
	require.Equal(t, "unspecified", ActionLabel(frontlinev1.Action_ACTION_UNSPECIFIED))
}

// Execute should not touch the incoming request or HTTP response — it's a
// pure config evaluator. Regression guard for future changes.
func TestFirewallExecutor_DoesNotTouchRequest(t *testing.T) {
	t.Parallel()

	e := &Executor{}
	req := httptest.NewRequest("GET", "/xxx", nil)
	req.Header.Set("X-Original", "yes")
	_, _ = e.Execute(context.Background(), nil, req, &frontlinev1.Firewall{ //nolint:exhaustruct
		Action: frontlinev1.Action_ACTION_DENY,
	})
	require.Equal(t, "yes", req.Header.Get("X-Original"))
	require.Equal(t, http.MethodGet, req.Method)
}
