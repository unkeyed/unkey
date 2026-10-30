package openapi

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
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

// decodeExpansionMax is the worst-case heap the validator uses per body byte.
// Measured peak heap for bodies like [0,0,...] or [{},{},...] was ~60x the body size
const decodeExpansionMax = 64

// bodyReadTimeout limits how long a slow client can hold its body reservation.
// A body at the default 4MiB max needs an upload of at least ~1.1 Mbit/s
const bodyReadTimeout = 30 * time.Second

type Executor struct {
	cache        cache.Cache[string, *compiledSpec]
	bodyBytesMax int64
	bodyBudget   *semaphore.Weighted
}

func New(clk clock.Clock, bodyBytesMax, bodyBytesBudget int64) (*Executor, error) {
	if bodyBytesMax > 0 && bodyBytesBudget > 0 && bodyBytesBudget < decodeExpansionMax*bodyBytesMax {
		return nil, fmt.Errorf("openapi body budget %d is smaller than %d times the body limit %d", bodyBytesBudget, decodeExpansionMax, bodyBytesMax)
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
	sess *zen.Session,
	req *http.Request,
	cfg *frontlinev1.OpenApiRequestValidation,
) (*redaction.Redactor, error) {
	spec := cfg.GetSpecYaml()
	if len(spec) == 0 {
		return nil, nil
	}

	if err := e.limitBody(sess.ResponseWriter(), req); err != nil {
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

	// limitBody already reserved the body itself, so reserve one byte per body byte less
	decodeBytesReserved := (decodeExpansionMax - 1) * req.ContentLength
	if e.bodyBudget != nil && e.bodyBytesMax > 0 && decodeBytesReserved > 0 {
		if !e.bodyBudget.TryAcquire(decodeBytesReserved) {
			return nil, validationBusy(decodeBytesReserved)
		}
		defer e.bodyBudget.Release(decodeBytesReserved)
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

// limitBody reads the body into memory, up to bodyBytesMax, because the validator
// reads it with no limit. The body stays reserved until the request ends, because
// it stays in memory until the proxy forwards it. A body without Content-Length
// reserves bodyBytesMax, then gives back what it did not use.
//
// It runs before Execute reserves decode memory, so a slow client cannot hold
// that reservation while it sends bytes. A set GetBody means an earlier OpenAPI
// policy on this request already read and reserved the body
func (e *Executor) limitBody(w http.ResponseWriter, req *http.Request) (err error) {
	if e.bodyBytesMax <= 0 || req.Body == nil || req.Body == http.NoBody || req.GetBody != nil {
		return nil
	}
	if req.ContentLength > e.bodyBytesMax {
		return bodyTooLarge(e.bodyBytesMax)
	}

	bytesReserved := req.ContentLength
	if bytesReserved < 0 {
		bytesReserved = e.bodyBytesMax
	}
	bytesHeld := bytesReserved
	if e.bodyBudget != nil && bytesReserved > 0 {
		if !e.bodyBudget.TryAcquire(bytesReserved) {
			return validationBusy(bytesReserved)
		}
		defer func() {
			if err != nil {
				bytesHeld = 0
			}
			e.bodyBudget.Release(bytesReserved - bytesHeld)
			if bytesHeld > 0 {
				context.AfterFunc(req.Context(), func() { e.bodyBudget.Release(bytesHeld) })
			}
		}()
	}

	rc := http.NewResponseController(w)
	if err := rc.SetReadDeadline(time.Now().Add(bodyReadTimeout)); err != nil {
		return fault.Wrap(err,
			fault.Code(codes.Frontline.Internal.InternalServerError.URN()),
			fault.Internal("unable to set request body read deadline"),
		)
	}
	defer func() {
		if clearErr := rc.SetReadDeadline(time.Time{}); clearErr != nil {
			err = errors.Join(err, fault.Wrap(clearErr,
				fault.Code(codes.Frontline.Internal.InternalServerError.URN()),
				fault.Internal("unable to clear request body read deadline"),
			))
		}
	}()

	body, err := io.ReadAll(io.LimitReader(req.Body, e.bodyBytesMax+1))
	closeErr := req.Body.Close()
	if errors.Is(err, os.ErrDeadlineExceeded) {
		return fault.Wrap(err,
			fault.Code(codes.User.BadRequest.RequestBodyUnreadable.URN()),
			fault.Internal("request body read deadline exceeded"),
			fault.Public(fmt.Sprintf("The request body did not arrive within %s.", bodyReadTimeout)),
		)
	}
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
	bytesHeld = min(int64(cap(body)), bytesReserved)

	req.Body = io.NopCloser(bytes.NewReader(body))
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}
	req.ContentLength = int64(len(body))
	return nil
}

func validationBusy(bytesWanted int64) error {
	return fault.New("openapi body budget exhausted",
		fault.Code(codes.Frontline.OpenApi.ValidationBusy.URN()),
		fault.Internal(fmt.Sprintf("no budget left to hold %d bytes for validation", bytesWanted)),
		fault.Public("The gateway is validating too many request bodies right now. Retry shortly."),
	)
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
