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
	openapiExec "github.com/unkeyed/unkey/svc/frontline/internal/policies/openapi"
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
	Evaluate(ctx context.Context, sess *zen.Session, req *http.Request, workspaceID string, mw []*frontlinev1.Policy) (Result, error)
}

// Engine implements Evaluator.
type Engine struct {
	keyAuth     *keyauthExec.Executor
	rateLimiter *ratelimitExec.Executor
	firewall    *firewallExec.Executor
	openapi     *openapiExec.Executor
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
	openapi, err := openapiExec.New(cfg.Clock)
	if err != nil {
		return nil, fmt.Errorf("create openapi executor: %w", err)
	}

	return &Engine{
		keyAuth:     keyauthExec.New(cfg.KeyService, cfg.Clock),
		rateLimiter: ratelimitExec.New(cfg.RateLimiter, cfg.Clock),
		firewall:    firewallExec.New(),
		openapi:     openapi,
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
	workspaceID string,
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
			outcome := "success"
			if execErr != nil {
				outcome = classifyKeyauthError(execErr)
			}
			engineEvaluationDuration.WithLabelValues("keyauth", outcome).Observe(time.Since(t).Seconds())
			engineEvaluationsTotal.WithLabelValues("keyauth", outcome).Inc()

			if execErr != nil {
				return result, execErr
			}

			if principal != nil {
				result.Principal = principal
			}

		case *frontlinev1.Policy_Ratelimit:
			t := time.Now()
			execErr := e.rateLimiter.Execute(ctx, sess, req, workspaceID, policy.GetId(), cfg.Ratelimit, result.Principal)
			outcome := "success"
			if execErr != nil {
				outcome = classifyRatelimitError(execErr)
			}
			engineEvaluationDuration.WithLabelValues("ratelimit", outcome).Observe(time.Since(t).Seconds())
			engineEvaluationsTotal.WithLabelValues("ratelimit", outcome).Inc()

			if execErr != nil {
				return result, execErr
			}

		case *frontlinev1.Policy_Firewall:
			t := time.Now()
			action, execErr := e.firewall.Execute(ctx, sess, req, cfg.Firewall)
			outcome := "noop"
			if execErr != nil {
				outcome = classifyFirewallError(execErr)
			}
			engineEvaluationDuration.WithLabelValues("firewall", outcome).Observe(time.Since(t).Seconds())
			firewallMatchesTotal.WithLabelValues(policy.GetId(), firewallExec.ActionLabel(action)).Inc()
			engineEvaluationsTotal.WithLabelValues("firewall", outcome).Inc()

			if execErr != nil {
				return result, execErr
			}

		case *frontlinev1.Policy_Openapi:
			t := time.Now()
			execErr := e.openapi.Execute(ctx, sess, req, cfg.Openapi)
			outcome := "success"
			if execErr != nil {
				outcome = "rejected"
			}
			engineEvaluationDuration.WithLabelValues("openapi", outcome).Observe(time.Since(t).Seconds())
			engineEvaluationsTotal.WithLabelValues("openapi", outcome).Inc()

			if execErr != nil {
				return result, execErr
			}

		default:
			continue
		}
	}

	return result, nil
}
