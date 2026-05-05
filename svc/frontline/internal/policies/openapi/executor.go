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
	"github.com/unkeyed/unkey/pkg/zen"
)

type Executor struct {
	cache cache.Cache[string, *validation.Validator]
}

func New(clk clock.Clock) (*Executor, error) {
	c, err := cache.New(cache.Config[string, *validation.Validator]{
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
) error {
	spec := cfg.GetSpecYaml()
	if len(spec) == 0 {
		return nil
	}

	v, err := e.getOrCompile(ctx, spec)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.Frontline.Internal.InvalidConfiguration.URN()),
			fault.Internal("failed to compile OpenAPI spec"),
			fault.Public("Service configuration error"),
		)
	}

	result := v.Validate(req)
	if result == nil {
		return nil
	}

	publicMsg := result.Detail
	if len(result.Errors) > 0 {
		publicMsg = result.Detail + ": " + result.Errors[0].Message
	}

	return fault.New("request validation failed",
		fault.Code(codes.Frontline.OpenApi.InvalidRequest.URN()),
		fault.Internal(publicMsg),
		fault.Public(publicMsg),
	)
}

// getOrCompile returns a compiled validator, using SWR cache keyed by spec content.
// Keying by content means deployments sharing the same spec reuse one compiled validator.
func (e *Executor) getOrCompile(ctx context.Context, spec []byte) (*validation.Validator, error) {
	v, _, err := e.cache.SWR(ctx, string(spec),
		func(ctx context.Context) (*validation.Validator, error) {
			return validation.NewFromBytes(spec)
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
