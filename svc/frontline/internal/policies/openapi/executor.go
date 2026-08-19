package openapi

import (
	"context"
	"net/http"
	"time"

	frontlinev1 "github.com/unkeyed/unkey/gen/proto/frontline/v1"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	validation "github.com/unkeyed/unkey/pkg/openapi/validation"
	"github.com/unkeyed/unkey/pkg/redaction"
	"github.com/unkeyed/unkey/pkg/zen"
)

type compiledSpec struct {
	validator *validation.Validator
	redactor  *redaction.Redactor
}

type Executor struct {
	cache cache.Cache[string, *compiledSpec]
}

func New(clk clock.Clock) (*Executor, error) {
	c, err := cache.New(cache.Config[string, *compiledSpec]{
		Fresh:    time.Hour,
		Stale:    24 * time.Hour,
		MaxSize:  64,
		Resource: "openapi_validators",
		Clock:    clk,
	})
	if err != nil {
		return nil, err
	}
	return &Executor{cache: c}, nil
}

func (e *Executor) Execute(
	ctx context.Context,
	_ *zen.Session,
	req *http.Request,
	cfg *frontlinev1.OpenApiRequestValidation,
) (*redaction.Redactor, error) {
	spec := cfg.GetSpecYaml()
	if len(spec) == 0 {
		return nil, nil
	}

	compiled, err := e.getOrCompile(ctx, spec)
	if err != nil {
		return nil, fault.Wrap(err,
			fault.Code(codes.Frontline.Internal.InvalidConfiguration.URN()),
			fault.Internal("failed to compile OpenAPI spec"),
			fault.Public("Service configuration error"),
		)
	}

	result := compiled.validator.Validate(req)
	if result == nil {
		return compiled.redactor, nil
	}

	publicMsg := result.Detail
	if len(result.Errors) > 0 {
		publicMsg = result.Detail + ": " + result.Errors[0].Message
	}

	return nil, fault.New("request validation failed",
		fault.Code(codes.Frontline.OpenApi.InvalidRequest.URN()),
		fault.Internal(publicMsg),
		fault.Public(publicMsg),
	)
}

// getOrCompile returns a compiled validator and body redactor, using SWR cache keyed by spec content.
// Keying by content means deployments sharing the same spec reuse one compiled validator.
func (e *Executor) getOrCompile(ctx context.Context, spec []byte) (*compiledSpec, error) {
	v, _, err := e.cache.SWR(ctx, string(spec),
		func(ctx context.Context) (*compiledSpec, error) {
			validator, compileErr := validation.NewFromBytes(spec)
			if compileErr != nil {
				return nil, compileErr
			}
			paths, compileErr := redaction.PathsFromSpec(spec)
			if compileErr != nil {
				return nil, compileErr
			}
			var bodyRedactor *redaction.Redactor
			if len(paths) > 0 {
				bodyRedactor = redaction.New(paths)
			}
			return &compiledSpec{
				validator: validator,
				redactor:  bodyRedactor,
			}, nil
		},
		func(err error) cache.Op {
			if err != nil {
				return cache.Noop
			}
			return cache.WriteValue
		},
	)
	return v, err
}
