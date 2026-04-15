package engine

import (
	"context"
	"fmt"
	"net/http"
	"time"

	sentinelv1 "github.com/unkeyed/unkey/gen/proto/sentinel/v1"
	"github.com/unkeyed/unkey/internal/services/keys"
	rl "github.com/unkeyed/unkey/internal/services/ratelimit"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/zen"
	firewallExec "github.com/unkeyed/unkey/svc/sentinel/engine/firewall"
	keyauthExec "github.com/unkeyed/unkey/svc/sentinel/engine/keyauth"
	"github.com/unkeyed/unkey/svc/sentinel/engine/principal"
	ratelimitExec "github.com/unkeyed/unkey/svc/sentinel/engine/ratelimit"
	"google.golang.org/protobuf/encoding/protojson"
)

// PrincipalHeader is the header name used to pass the authenticated principal
// to upstream services.
const PrincipalHeader = "X-Unkey-Principal"

// Config holds the configuration for creating a new Engine.
type Config struct {
	KeyService  keys.KeyService
	RateLimiter rl.Service
	Clock       clock.Clock
}

// Evaluator evaluates sentinel middleware policies against incoming requests.
type Evaluator interface {
	Evaluate(ctx context.Context, sess *zen.Session, req *http.Request, mw []*sentinelv1.Policy) (Result, error)
}

// Engine implements Evaluator.
type Engine struct {
	keyAuth     *keyauthExec.Executor
	rateLimiter *ratelimitExec.Executor
	firewall    *firewallExec.Executor
	regexCache  *regexCache
}

var _ Evaluator = (*Engine)(nil)

// Result holds the outcome of middleware evaluation.
type Result struct {
	Principal *principal.Principal
}

// New creates a new Engine with the given configuration.
func New(cfg Config) (*Engine, error) {
	if err := assert.All(
		assert.NotNil(cfg.KeyService, "cfg.KeyService must not be nil"),
		assert.NotNil(cfg.RateLimiter, "cfg.RateLimiter must not be nil"),
		assert.NotNil(cfg.Clock, "cfg.Clock must not be nil"),
	); err != nil {
		return nil, err
	}
	return &Engine{
		keyAuth:     keyauthExec.New(cfg.KeyService, cfg.Clock),
		rateLimiter: ratelimitExec.New(cfg.RateLimiter, cfg.Clock),
		firewall:    firewallExec.New(),
		regexCache:  newRegexCache(),
	}, nil
}

// ParseMiddleware performs lenient deserialization of sentinel_config bytes into
// a Middleware proto. Returns nil for empty, legacy empty-object, or malformed data
// to allow plain pass-through proxying.
func ParseMiddleware(raw []byte) ([]*sentinelv1.Policy, error) {
	if len(raw) == 0 || string(raw) == "{}" {
		return nil, nil
	}

	cfg := &sentinelv1.Config{}
	unmarshalOpts := protojson.UnmarshalOptions{DiscardUnknown: true}
	if err := unmarshalOpts.Unmarshal(raw, cfg); err != nil {
		return nil, fault.Wrap(err,
			fault.Code(codes.Sentinel.Internal.InvalidConfiguration.URN()),
			fault.Internal(fmt.Sprintf("unable to unmarshal sentinel policies: %s", string(raw))),
			fault.Public("The policy configuration is invalid. Please check your sentinel config or contact support at support@unkey.com."),
		)
	}

	if len(cfg.GetPolicies()) == 0 {
		return nil, nil
	}

	return cfg.GetPolicies(), nil
}

// Evaluate processes all middleware policies against the incoming request.
// Policies are evaluated in order. Disabled policies are skipped.
// Authentication policies produce a Principal; the first successful auth sets it.
//
// Firewall policies short-circuit the request with a Firewall.Denied fault
// when their match expressions hit and the action is ACTION_DENY. The
// action enum exists for forward compatibility — additional outcomes will
// be wired into the dispatch when they're added to the proto.
func (e *Engine) Evaluate(
	ctx context.Context,
	sess *zen.Session,
	req *http.Request,
	policies []*sentinelv1.Policy,
) (Result, error) {
	var result Result

	for _, policy := range policies {
		if !policy.GetEnabled() {
			continue
		}

		// Check match expressions
		matched, err := matchesRequest(req, policy.GetMatch(), e.regexCache)
		if err != nil {
			return result, err
		}

		if !matched {
			continue
		}

		// Dispatch by config type
		switch cfg := policy.GetConfig().(type) {
		case *sentinelv1.Policy_Keyauth:
			// Skip if we already have a principal from a previous auth policy
			if result.Principal != nil {
				sentinelEngineEvaluationsTotal.WithLabelValues("keyauth", "skipped").Inc()
				continue
			}

			t := time.Now()
			principal, execErr := e.keyAuth.Execute(ctx, sess, req, cfg.Keyauth)
			sentinelEngineEvaluationDuration.WithLabelValues("keyauth").Observe(time.Since(t).Seconds())

			if execErr != nil {
				sentinelEngineEvaluationsTotal.WithLabelValues("keyauth", classifyKeyauthError(execErr)).Inc()
				return result, execErr
			}

			if principal != nil {
				result.Principal = principal
				sentinelEngineEvaluationsTotal.WithLabelValues("keyauth", "success").Inc()
			}

		case *sentinelv1.Policy_Ratelimit:
			t := time.Now()
			execErr := e.rateLimiter.Execute(ctx, sess, req, policy.GetId(), cfg.Ratelimit, result.Principal)
			sentinelEngineEvaluationDuration.WithLabelValues("ratelimit").Observe(time.Since(t).Seconds())

			if execErr != nil {
				sentinelEngineEvaluationsTotal.WithLabelValues("ratelimit", classifyRatelimitError(execErr)).Inc()
				return result, execErr
			}

			sentinelEngineEvaluationsTotal.WithLabelValues("ratelimit", "success").Inc()
		case *sentinelv1.Policy_Firewall:
			t := time.Now()
			action, execErr := e.firewall.Execute(ctx, sess, req, cfg.Firewall)
			sentinelEngineEvaluationDuration.WithLabelValues("firewall").Observe(time.Since(t).Seconds())

			sentinelFirewallMatchesTotal.WithLabelValues(policy.GetId(), firewallExec.ActionLabel(action)).Inc()

			if execErr != nil {
				sentinelEngineEvaluationsTotal.WithLabelValues("firewall", classifyFirewallError(execErr)).Inc()
				return result, execErr
			}

			// ACTION_UNSPECIFIED (or unknown) — nothing to do.
			sentinelEngineEvaluationsTotal.WithLabelValues("firewall", "noop").Inc()

		// Future policy types will be added here:
		// case *sentinelv1.Policy_Jwtauth:
		// case *sentinelv1.Policy_Ratelimit:
		// case *sentinelv1.Policy_Openapi:

		default:
			// Unknown policy type — skip silently for forward compatibility
			continue
		}
	}

	return result, nil
}
