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
	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/redaction"
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
	KeyService       keys.KeyService
	RateLimiter      rl.Service
	Clock            clock.Clock
	KeyVerifications *batch.BatchProcessor[schema.KeyVerification]
	OpenAPIExecutor  OpenAPIExecutor
}

// Evaluator evaluates policies against incoming requests.
type Evaluator interface {
	Evaluate(ctx context.Context, sess *zen.Session, req *http.Request, workspaceID, appID string, mw []*frontlinev1.Policy) (Result, error)
}

// OpenAPIExecutor validates a request against an OpenAPI policy.
type OpenAPIExecutor interface {
	Execute(ctx context.Context, sess *zen.Session, req *http.Request, cfg *frontlinev1.OpenApiRequestValidation) (*redaction.Redactor, error)
}

// Engine implements Evaluator.
type Engine struct {
	keyAuth     *keyauthExec.Executor
	rateLimiter *ratelimitExec.Executor
	firewall    *firewallExec.Executor
	openapi     OpenAPIExecutor
	regexCache  *regexCache
}

var _ Evaluator = (*Engine)(nil)

// Result holds the outcome of policy evaluation.
type Result struct {
	Principal     *principal.Principal
	BodyRedactors []*redaction.Redactor

	// Capture flags set by matching enabled logging policies. Each is a
	// separate opt-in; the base ClickHouse row is always written regardless
	// of logging policies. LogRequestHeaders also covers the user agent and
	// client IP; LogQuery covers the query string and query parameters.
	LogRequestHeaders  bool
	LogResponseHeaders bool
	LogRequestBody     bool
	LogResponseBody    bool
	LogQuery           bool
}

// New creates a new Engine with the given configuration.
func New(cfg Config) (*Engine, error) {
	if err := assert.All(
		assert.NotNil(cfg.KeyService, "cfg.KeyService must not be nil"),
		assert.NotNil(cfg.RateLimiter, "cfg.RateLimiter must not be nil"),
		assert.NotNil(cfg.Clock, "cfg.Clock must not be nil"),
		assert.NotNil(cfg.KeyVerifications, "cfg.KeyVerifications must not be nil"),
		assert.NotNil(cfg.OpenAPIExecutor, "cfg.OpenAPIExecutor must not be nil"),
	); err != nil {
		return nil, err
	}

	return &Engine{
		keyAuth:     keyauthExec.New(cfg.KeyService, cfg.Clock, cfg.KeyVerifications),
		rateLimiter: ratelimitExec.New(cfg.RateLimiter, cfg.Clock),
		firewall:    firewallExec.New(),
		openapi:     cfg.OpenAPIExecutor,
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
	appID string,
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
			principal, execErr := e.keyAuth.Execute(ctx, sess, req, appID, cfg.Keyauth)
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
			execErr := e.rateLimiter.Execute(ctx, sess, req, workspaceID, policy.GetId(), cfg.Ratelimit, result.Principal)
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

		case *frontlinev1.Policy_Openapi:
			t := time.Now()
			bodyRedactor, execErr := e.openapi.Execute(ctx, sess, req, cfg.Openapi)
			engineEvaluationDuration.WithLabelValues("openapi").Observe(time.Since(t).Seconds())

			if execErr != nil {
				engineEvaluationsTotal.WithLabelValues("openapi", "rejected").Inc()
				return result, execErr
			}
			if bodyRedactor != nil {
				result.BodyRedactors = append(result.BodyRedactors, bodyRedactor)
			}

			engineEvaluationsTotal.WithLabelValues("openapi", "success").Inc()

		case *frontlinev1.Policy_Logging:
			// Logging is observational, not an enforcement action: a matching
			// enabled policy opts the request into capturing headers and/or
			// bodies in the ClickHouse request log. The base row is written
			// unconditionally by the logging middleware. Multiple matching
			// policies OR their capture flags. The actual capture and
			// emission happen in the handler and the ClickHouse logging
			// middleware.
			result.LogRequestHeaders = result.LogRequestHeaders || cfg.Logging.GetRequestHeaders()
			result.LogResponseHeaders = result.LogResponseHeaders || cfg.Logging.GetResponseHeaders()
			result.LogRequestBody = result.LogRequestBody || cfg.Logging.GetRequestBody()
			result.LogResponseBody = result.LogResponseBody || cfg.Logging.GetResponseBody()
			result.LogQuery = result.LogQuery || cfg.Logging.GetQuery()
			engineEvaluationsTotal.WithLabelValues("logging", "success").Inc()

		default:
			continue
		}
	}

	return result, nil
}
