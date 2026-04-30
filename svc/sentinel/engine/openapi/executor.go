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

// Specs rarely change -- a service typically deploys with one and keeps it for weeks.
// We cache compiled validators keyed by spec content and cap at maxValidators as a
// safety net. When the cap is hit we wipe and recompile, which is cheap given how infrequently specs actually change.
const maxValidators = 64

type Executor struct {
	mu         sync.RWMutex
	validators map[string]*validation.Validator
	size       int
}

func New() *Executor {
	return &Executor{
		mu:         sync.RWMutex{},
		validators: make(map[string]*validation.Validator),
		size:       0,
	}
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

	e.mu.RLock()
	v, ok := e.validators[key]
	e.mu.RUnlock()
	if ok {
		return v, nil
	}

	compiled, err := validation.NewFromBytes(spec)
	if err != nil {
		return nil, err
	}

	e.mu.Lock()
	if e.size >= maxValidators {
		e.validators = make(map[string]*validation.Validator)
		e.size = 0
	}
	e.validators[key] = compiled
	e.size++
	e.mu.Unlock()

	return compiled, nil
}
