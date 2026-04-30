package policies

import (
	"context"
	"fmt"
	"net/http"
	"time"

	frontlinev1 "github.com/unkeyed/unkey/gen/proto/frontline/v1"
	"github.com/unkeyed/unkey/internal/services/keys"
	rl "github.com/unkeyed/unkey/internal/services/ratelimit"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/zen"
	firewallExec "github.com/unkeyed/unkey/svc/frontline/internal/policies/firewall"
	keyauthExec "github.com/unkeyed/unkey/svc/frontline/internal/policies/keyauth"
	"github.com/unkeyed/unkey/svc/frontline/internal/policies/principal"
	ratelimitExec "github.com/unkeyed/unkey/svc/frontline/internal/policies/ratelimit"
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

// Evaluator evaluates policies against incoming requests.
type Evaluator interface {
	Evaluate(ctx context.Context, sess *zen.Session, req *http.Request, mw []*frontlinev1.Policy) (Result, error)
}

// Engine implements Evaluator.
type Engine struct {
	keyAuth     *keyauthExec.Executor
	rateLimiter *ratelimitExec.Executor
	firewall    *firewallExec.Executor
	regexCache  *regexCache
}

var _ Evaluator = (*Engine)(nil)

// Result holds the outcome of policy evaluation.
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
func ParseMiddleware(raw []byte) ([]*frontlinev1.Policy, error) {
	if len(raw) == 0 || string(raw) == "{}" {
		return nil, nil
	}

	cfg := &frontlinev1.Config{}
	unmarshalOpts := protojson.UnmarshalOptions{DiscardUnknown: true}
	if err := unmarshalOpts.Unmarshal(raw, cfg); err != nil {
		return nil, fault.Wrap(err,
			fault.Code(codes.Frontline.Internal.InvalidConfiguration.URN()),
			fault.Internal(fmt.Sprintf("unable to unmarshal policies: %s", string(raw))),
			fault.Public("The policy configuration is invalid. Please check your config or contact support at support@unkey.com."),
		)
	}

	if len(cfg.GetPolicies()) == 0 {
		return nil, nil
	}

	return cfg.GetPolicies(), nil
}

// Evaluate processes all policies against the incoming request.
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
	policies []*frontlinev1.Policy,
) (Result, error) {
	var result Result

	for _, policy := range policies {
		if !policy.GetEnabled() {
			continue
		}

		matched, err := matchesRequest(req, policy.GetMatch(), e.regexCache)
		if err != nil {
			return result, err
		}

		if !matched {
			continue
		}

		switch cfg := policy.GetConfig().(type) {
		case *frontlinev1.Policy_Keyauth:
			if result.Principal != nil {
				engineEvaluationsTotal.WithLabelValues("keyauth", "skipped").Inc()
				continue
			}

			t := time.Now()
			principal, execErr := e.keyAuth.Execute(ctx, sess, req, cfg.Keyauth)
			engineEvaluationDuration.WithLabelValues("keyauth").Observe(time.Since(t).Seconds())

			if execErr != nil {
				engineEvaluationsTotal.WithLabelValues("keyauth", classifyKeyauthError(execErr)).Inc()
				return result, execErr
			}

			if principal != nil {
				result.Principal = principal
				engineEvaluationsTotal.WithLabelValues("keyauth", "success").Inc()
			}

		case *frontlinev1.Policy_Ratelimit:
			t := time.Now()
			execErr := e.rateLimiter.Execute(ctx, sess, req, policy.GetId(), cfg.Ratelimit, result.Principal)
			engineEvaluationDuration.WithLabelValues("ratelimit").Observe(time.Since(t).Seconds())

			if execErr != nil {
				engineEvaluationsTotal.WithLabelValues("ratelimit", classifyRatelimitError(execErr)).Inc()
				return result, execErr
			}

			engineEvaluationsTotal.WithLabelValues("ratelimit", "success").Inc()
		case *frontlinev1.Policy_Firewall:
			t := time.Now()
			action, execErr := e.firewall.Execute(ctx, sess, req, cfg.Firewall)
			engineEvaluationDuration.WithLabelValues("firewall").Observe(time.Since(t).Seconds())

			firewallMatchesTotal.WithLabelValues(policy.GetId(), firewallExec.ActionLabel(action)).Inc()

			if execErr != nil {
				engineEvaluationsTotal.WithLabelValues("firewall", classifyFirewallError(execErr)).Inc()
				return result, execErr
			}

			engineEvaluationsTotal.WithLabelValues("firewall", "noop").Inc()

		default:
			continue
		}
	}

	return result, nil
}
