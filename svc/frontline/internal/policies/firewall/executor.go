package firewall

import (
	"context"
	"net/http"

	frontlinev1 "github.com/unkeyed/unkey/gen/proto/frontline/v1"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/zen"
)

// Executor applies a [Firewall] policy's action to a matched request.
// It is intentionally stateless and pure: no network I/O, no database access.
// Match semantics live in match.go; this executor only decides what to return
// once a match has already succeeded. Safe for concurrent use by design —
// there is no mutable state to share.
type Executor struct{}

// New creates a new Firewall policy executor.
func New() *Executor {
	return &Executor{}
}

// Execute applies the firewall action. ACTION_DENY returns a fault under
// the Sentinel.Firewall.Denied URN, which the middleware layer translates
// to a 403 response with a fixed "Forbidden" body. Unspecified or unknown
// action values are treated as a no-op for forward compatibility.
func (e *Executor) Execute(
	_ context.Context,
	_ *zen.Session,
	_ *http.Request,
	cfg *frontlinev1.Firewall,
) (frontlinev1.Action, error) {
	action := cfg.GetAction()
	//nolint:exhaustive
	switch action {
	case frontlinev1.Action_ACTION_DENY:
		return action, fault.New("firewall denied",
			fault.Code(codes.Frontline.Firewall.Denied.URN()),
			fault.Internal("request denied by Firewall policy"),
			fault.Public("Forbidden"),
		)

	default:
		return frontlinev1.Action_ACTION_UNSPECIFIED, nil
	}
}

// ActionLabel returns the metric label for a firewall action. Kept
// separate from the proto String() so labels stay stable even if proto enum
// names change.
func ActionLabel(a frontlinev1.Action) string {
	//nolint:exhaustive
	switch a {
	case frontlinev1.Action_ACTION_DENY:
		return "deny"
	default:
		return "unspecified"
	}
}
