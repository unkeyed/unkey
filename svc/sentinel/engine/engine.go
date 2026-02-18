package engine

import (
	"context"
	"net/http"

	sentinelv1 "github.com/unkeyed/unkey/gen/proto/sentinel/v1"
	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/zen"
	"google.golang.org/protobuf/encoding/protojson"
)

// PrincipalHeader is the header name used to pass the authenticated principal
// to upstream services.
const PrincipalHeader = "X-Unkey-Principal"

// Config holds the configuration for creating a new Engine.
type Config struct {
	KeyService keys.KeyService
	Clock      clock.Clock
}

// Evaluator evaluates sentinel middleware policies against incoming requests.
type Evaluator interface {
	Evaluate(ctx context.Context, sess *zen.Session, req *http.Request, mw *sentinelv1.Middleware) (Result, error)
}

// Engine implements Evaluator.
type Engine struct {
	keyAuth    *KeyAuthExecutor
	regexCache *regexCache
}

var _ Evaluator = (*Engine)(nil)

// Result holds the outcome of middleware evaluation.
type Result struct {
	Principal *sentinelv1.Principal
}

// New creates a new Engine with the given configuration.
func New(cfg Config) *Engine {
	return &Engine{
		keyAuth: &KeyAuthExecutor{
			keyService: cfg.KeyService,
			clock:      cfg.Clock,
		},
		regexCache: newRegexCache(),
	}
}

// ParseMiddleware performs lenient deserialization of sentinel_config bytes into
// a Middleware proto. Returns nil for empty, legacy empty-object, or malformed data
// to allow plain pass-through proxying.
func ParseMiddleware(raw []byte) *sentinelv1.Middleware {
	if len(raw) == 0 || string(raw) == "{}" {
		return nil
	}

	mw := &sentinelv1.Middleware{}
	if err := protojson.Unmarshal(raw, mw); err != nil {
		logger.Warn("failed to unmarshal sentinel middleware config, treating as pass-through",
			"error", err.Error(),
		)

		return nil
	}

	if len(mw.GetPolicies()) == 0 {
		return nil
	}

	return mw
}

// Evaluate processes all middleware policies against the incoming request.
// Policies are evaluated in order. Disabled policies are skipped.
// Authentication policies produce a Principal; the first successful auth sets it.
func (e *Engine) Evaluate(
	ctx context.Context,
	sess *zen.Session,
	req *http.Request,
	mw *sentinelv1.Middleware,
) (Result, error) {
	var result Result

	for _, policy := range mw.GetPolicies() {
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
				continue
			}

			principal, execErr := e.keyAuth.Execute(ctx, sess, req, cfg.Keyauth)
			if execErr != nil {
				return result, execErr
			}

			if principal != nil {
				result.Principal = principal
			}

		// Future policy types will be added here:
		// case *sentinelv1.Policy_Jwtauth:
		// case *sentinelv1.Policy_Basicauth:
		// case *sentinelv1.Policy_Ratelimit:
		// case *sentinelv1.Policy_IpRules:
		// case *sentinelv1.Policy_Openapi:

		default:
			// Unknown policy type — skip silently for forward compatibility
			continue
		}
	}

	return result, nil
}

// SerializePrincipal converts a Principal to a JSON string for use in the
// X-Unkey-Principal header.
func SerializePrincipal(p *sentinelv1.Principal) (string, error) {
	b, err := protojson.Marshal(p)
	if err != nil {
		return "", err
	}

	return string(b), nil
}
