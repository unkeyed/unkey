package engine

import (
	"context"
	"net/http"

	sentinelv1 "github.com/unkeyed/unkey/gen/proto/sentinel/v1"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/zen"
)

// FirewallExecutor applies a [Firewall] policy's action to a matched request.
// It is intentionally stateless and pure: no network I/O, no database access.
// Match semantics live in match.go; this executor only decides what to return
// once a match has already succeeded. Safe for concurrent use by design —
// there is no mutable state to share.
type FirewallExecutor struct{}

// Execute applies the firewall action. ACTION_DENY returns a fault under
// the Sentinel.Firewall.Denied URN, which the middleware layer translates
// to a 403 response with a fixed "Forbidden" body. Unspecified or unknown
// action values are treated as a no-op for forward compatibility.
func (e *FirewallExecutor) Execute(
	_ context.Context,
	_ *zen.Session,
	_ *http.Request,
	cfg *sentinelv1.Firewall,
) (sentinelv1.Action, error) {
	action := cfg.GetAction()
	//nolint:exhaustive
	switch action {
	case sentinelv1.Action_ACTION_DENY:
		return action, fault.New("firewall denied",
			fault.Code(codes.Sentinel.Firewall.Denied.URN()),
			fault.Internal("request denied by Firewall policy"),
			fault.Public("Forbidden"),
		)

	default:
		return sentinelv1.Action_ACTION_UNSPECIFIED, nil
	}
}
