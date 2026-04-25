package edgeredirect

import (
	"net/http"

	edgeredirectv1 "github.com/unkeyed/unkey/gen/proto/frontline/edgeredirect/v1"
)

// defaultStatus is the HTTP code used when a Rule does not specify one.
// 308 Permanent Redirect preserves the request method, so clients do not
// silently downgrade POST/PUT/DELETE to GET as they would with 301/302.
const defaultStatus = http.StatusPermanentRedirect

// Result is what a successful evaluation returns. RuleID and RuleKind are
// included for caller-side metric labeling and log correlation; the engine
// itself emits no metrics so callers can pick the cardinality they want.
type Result struct {
	Location string
	Status   int
	RuleID   string
	RuleKind string
}

// Evaluator is the engine interface. The concrete *Engine is the only
// production implementation; the interface exists so consumers (the HTTP
// catchall handler, the HTTPS proxy handler) can be tested with a fake.
type Evaluator interface {
	Evaluate(req *http.Request, rules []*edgeredirectv1.Rule) *Result
}

// Engine evaluates rules. It carries no shared state today; the type
// exists as a seam for future fields (e.g. a regex cache) without touching
// every call site.
type Engine struct{}

// New constructs an Engine.
func New() *Engine { return &Engine{} }

var _ Evaluator = (*Engine)(nil)

// Evaluate walks rules in order and returns the first match. Returns nil
// when no enabled rule applies. Disabled rules are skipped without
// inspecting their kind, matching sentinel's behavior (svc/sentinel/engine).
func (e *Engine) Evaluate(req *http.Request, rules []*edgeredirectv1.Rule) *Result {
	if req == nil {
		return nil
	}

	for _, rule := range rules {
		if rule == nil || !rule.GetEnabled() {
			continue
		}

		location, ok := apply(req, rule)
		if !ok {
			continue
		}

		return &Result{
			Location: location,
			Status:   statusOrDefault(rule.GetStatus()),
			RuleID:   rule.GetId(),
			RuleKind: kindLabel(rule),
		}
	}

	return nil
}

func apply(req *http.Request, rule *edgeredirectv1.Rule) (string, bool) {
	switch rule.GetKind().(type) {
	case *edgeredirectv1.Rule_RequireHttps:
		return applyRequireHTTPS(req)
	case *edgeredirectv1.Rule_StripWww:
		return applyStripWWW(req)
	case *edgeredirectv1.Rule_AddWww:
		return applyAddWWW(req)
	case *edgeredirectv1.Rule_HostRewrite:
		return applyHostRewrite(req, rule.GetHostRewrite())
	default:
		// Unknown oneof — skip silently for forward compatibility with
		// future rule kinds that the binary does not yet understand.
		return "", false
	}
}

func statusOrDefault(s int32) int {
	if s == 0 {
		return defaultStatus
	}
	return int(s)
}

// kindLabel returns the metric/log label for a rule kind. Bounded
// cardinality (one per oneof option), safe for Prometheus labels.
func kindLabel(rule *edgeredirectv1.Rule) string {
	switch rule.GetKind().(type) {
	case *edgeredirectv1.Rule_RequireHttps:
		return "require_https"
	case *edgeredirectv1.Rule_StripWww:
		return "strip_www"
	case *edgeredirectv1.Rule_AddWww:
		return "add_www"
	case *edgeredirectv1.Rule_HostRewrite:
		return "host_rewrite"
	default:
		return "unknown"
	}
}
