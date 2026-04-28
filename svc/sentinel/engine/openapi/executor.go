package openapi

import (
	"context"
	"net/http"
	"sync"

	sentinelv1 "github.com/unkeyed/unkey/gen/proto/sentinel/v1"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	validation "github.com/unkeyed/unkey/pkg/openapi/validation"
	"github.com/unkeyed/unkey/pkg/zen"
)

type Executor struct {
	// spec content -> compiled validator, avoids re-parsing on every request
	validators sync.Map
}

func New() *Executor {
	//nolint:exhaustruct
	return &Executor{}
}

func (e *Executor) Execute(
	_ context.Context,
	_ *zen.Session,
	req *http.Request,
	cfg *sentinelv1.OpenApiRequestValidation,
) error {
	spec := cfg.GetSpecYaml()
	if len(spec) == 0 {
		return nil
	}

	v, err := e.getOrCompile(spec)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.Sentinel.Internal.InvalidConfiguration.URN()),
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
		fault.Code(codes.Sentinel.OpenApi.InvalidRequest.URN()),
		fault.Internal(publicMsg),
		fault.Public(publicMsg),
	)
}

func (e *Executor) getOrCompile(spec []byte) (*validation.Validator, error) {
	key := string(spec)

	if cached, ok := e.validators.Load(key); ok {
		return cached.(*validation.Validator), nil
	}

	compiled, err := validation.NewFromBytes(spec)
	if err != nil {
		return nil, err
	}

	actual, _ := e.validators.LoadOrStore(key, compiled)
	return actual.(*validation.Validator), nil
}
