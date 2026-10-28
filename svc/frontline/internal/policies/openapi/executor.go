package openapi

import (
	"bytes"
	"context"
	"fmt"
	"io"
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
	"golang.org/x/sync/semaphore"
)

type compiledSpec struct {
	validator *validation.Validator
	redactor  *redaction.Redactor
}

type Executor struct {
	cache        cache.Cache[string, *compiledSpec]
	bodyBytesMax int64
	bodyBudget   *semaphore.Weighted
}

// New creates an Executor. bodyBytesMax bounds one request body the validator
// reads, and bodyBytesBudget bounds the bytes all in-flight validations hold
// at once. Zero or less disables either bound.
func New(clk clock.Clock, bodyBytesMax, bodyBytesBudget int64) (*Executor, error) {
	if bodyBytesMax > 0 && bodyBytesBudget > 0 && bodyBytesBudget < bodyBytesMax {
		return nil, fmt.Errorf("openapi body budget %d is smaller than the body limit %d", bodyBytesBudget, bodyBytesMax)
	}
	var bodyBudget *semaphore.Weighted
	if bodyBytesBudget > 0 {
		bodyBudget = semaphore.NewWeighted(bodyBytesBudget)
	}

	c, err := cache.New(cache.Config[string, *compiledSpec]{
		Fresh:     time.Hour,
		Stale:     24 * time.Hour,
		MaxSize:   64,
		Resource:  "openapi_validators",
		Clock:     clk,
		Scheduler: nil,
	})
	if err != nil {
		return nil, err
	}
	return &Executor{cache: c, bodyBytesMax: bodyBytesMax, bodyBudget: bodyBudget}, nil
}

// Close releases the spec cache's background workers.
func (e *Executor) Close() {
	e.cache.Close()
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

	if err := e.capBody(req); err != nil {
		return nil, err
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

// A known-length body under the limit stays a stream, so duplex and streaming
// requests are not stalled. An unknown length must be buffered to bound what
// the validator's own read can consume.
//
// The validator restores the body it read onto the request, so the reserved
// bytes stay held until the request ends, not until validation returns.
func (e *Executor) capBody(req *http.Request) error {
	if e.bodyBytesMax <= 0 || req.Body == nil || req.Body == http.NoBody {
		return nil
	}
	if req.ContentLength > e.bodyBytesMax {
		return bodyTooLarge(e.bodyBytesMax)
	}

	bytesReserved := req.ContentLength
	if bytesReserved < 0 {
		bytesReserved = e.bodyBytesMax
	}
	if e.bodyBudget != nil && bytesReserved > 0 {
		if !e.bodyBudget.TryAcquire(bytesReserved) {
			return fault.New("openapi body budget exhausted",
				fault.Code(codes.Frontline.OpenApi.ValidationBusy.URN()),
				fault.Internal(fmt.Sprintf("no budget left to hold %d body bytes for validation", bytesReserved)),
				fault.Public("The gateway is validating too many request bodies right now. Retry shortly."),
			)
		}
		context.AfterFunc(req.Context(), func() { e.bodyBudget.Release(bytesReserved) })
	}

	if req.ContentLength >= 0 {
		return nil
	}

	body, err := io.ReadAll(io.LimitReader(req.Body, e.bodyBytesMax+1))
	closeErr := req.Body.Close()
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.User.BadRequest.RequestBodyUnreadable.URN()),
			fault.Internal("unable to read request body for validation"),
			fault.Public("The request body could not be read."),
		)
	}
	if closeErr != nil {
		return fault.Wrap(closeErr,
			fault.Code(codes.User.BadRequest.RequestBodyUnreadable.URN()),
			fault.Internal("failed to close request body"),
			fault.Public("The request body could not be read."),
		)
	}
	if int64(len(body)) > e.bodyBytesMax {
		return bodyTooLarge(e.bodyBytesMax)
	}

	req.Body = io.NopCloser(bytes.NewReader(body))
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}
	req.ContentLength = int64(len(body))
	return nil
}

func bodyTooLarge(bytesMax int64) error {
	return fault.New("request body too large",
		fault.Code(codes.User.BadRequest.RequestBodyTooLarge.URN()),
		fault.Internal(fmt.Sprintf("request body exceeds %d bytes", bytesMax)),
		fault.Public(fmt.Sprintf("The request body exceeds the maximum allowed size of %d bytes.", bytesMax)),
	)
}

// BodyRedactor returns the spec's redactor without validating or reading the request.
func (e *Executor) BodyRedactor(ctx context.Context, cfg *frontlinev1.OpenApiRequestValidation) (*redaction.Redactor, error) {
	if len(cfg.GetSpecYaml()) == 0 {
		return nil, nil
	}
	compiled, err := e.getOrCompile(ctx, cfg.GetSpecYaml())
	if err != nil {
		return nil, err
	}
	return compiled.redactor, nil
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
