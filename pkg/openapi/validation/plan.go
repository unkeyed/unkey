package validation

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"
	"sync"

	"github.com/pb33f/libopenapi"
	upstream "github.com/pb33f/libopenapi-validator"
	"github.com/pb33f/libopenapi-validator/config"
	"github.com/pb33f/libopenapi-validator/router"
	"github.com/pb33f/libopenapi-validator/schema_validation"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/openapi/streamschema"
)

// The pinned evaluator snapshots input and grows read/decoder buffers. Reserve
// copy headroom separately from decoded objects and garbage awaiting collection.
const bufferedCopyFactor = 8

// Plan selects streaming JSON validation where supported and the existing
// evaluator otherwise. Plans are immutable and safe for concurrent requests.
// This prototype is not wired into Frontline. It preserves existing non-JSON
// and format policies; it does not establish complete OpenAPI conformance.
type Plan struct {
	request *Validator
	full    *Validator
	routes  router.Router
	streams map[*v3.MediaType]*streamschema.Schema
	syntax  *streamschema.Schema
}

// Compile checks the OpenAPI document and prepares its request validation plan.
func Compile(spec []byte) (*Plan, error) {
	full, err := NewFromBytes(spec)
	if err != nil {
		return nil, err
	}
	document, err := libopenapi.NewDocument(spec)
	if err != nil {
		return nil, err
	}
	metadata, errs := upstream.NewValidator(document, config.WithRegexCache(&sync.Map{}), config.WithoutRequestBodyValidation())
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	model, err := document.BuildV3Model()
	if err != nil {
		return nil, err
	}
	syntax, err := streamschema.Compile([]byte("true"), streamschema.DefaultLimits())
	if err != nil {
		return nil, err
	}
	p := &Plan{
		request: &Validator{validator: metadata},
		full:    full,
		routes:  router.NewRouter(&model.Model, router.WithPathOnlyMatching()),
		streams: make(map[*v3.MediaType]*streamschema.Schema),
		syntax:  syntax,
	}
	if !strings.HasPrefix(model.Model.Version, "3.1.") || model.Model.Paths == nil {
		return p, nil
	}
	for path := model.Model.Paths.PathItems.First(); path != nil; path = path.Next() {
		for operation := path.Value().GetOperations().First(); operation != nil; operation = operation.Next() {
			body := operation.Value().RequestBody
			if body == nil || body.Content == nil {
				continue
			}
			media := body.Content.GetOrZero("application/json")
			if media == nil || media.Schema == nil {
				continue
			}
			rendered, renderErr := schema_validation.RenderSchemaForValidation(media.Schema.Schema(), schema_validation.SchemaValidationPurposeRequestBody)
			if renderErr != nil || rendered == nil {
				continue
			}
			compiled, compileErr := streamschema.Compile(rendered.RenderedJSON, streamschema.DefaultLimits())
			if compileErr == nil {
				p.streams[media] = compiled
			}
		}
	}
	return p, nil
}

// Validate owns req.Body and closes the input on every return. Success replaces
// it with a replayable body whose Close releases the budget; GetBody borrows
// readers valid until that Close. Failure installs http.NoBody. Result describes
// invalid input; error describes capacity, size, cancellation, or I/O failures.
//
// JSON applies streamschema's strict parsing policy and DefaultLimits before
// either evaluator. Malformed Content-Type parameters are rejected. Buffered
// evaluation still uses the existing decoder's numeric representation and
// allocations. Zero ContentLength with a body means unknown length on client
// requests, as in net/http; server requests have a nonempty RequestURI.
// Callers must set network read deadlines: closing a server body can wait for a
// blocked Read. Cancellation does not interrupt CPU work in the legacy evaluator.
func (p *Plan) Validate(req *http.Request, budget *Budget) (result *Result, err error) {
	if req == nil || req.Body == nil || budget == nil {
		return nil, fault.New("request, body, and budget must be non-nil")
	}
	input := req.Body
	var closeErr error
	closeInput := sync.OnceFunc(func() { closeErr = input.Close() })
	ctx := req.Context()
	callbackDone := make(chan struct{})
	stop := context.AfterFunc(ctx, func() {
		defer close(callbackDone)
		closeInput()
	})
	var body *retainedBody
	finished := false
	defer func() {
		closeInput()
		if !stop() {
			<-callbackDone
		}
		err = errors.Join(err, closeErr, ctx.Err())
		req.Body = http.NoBody
		req.GetBody = nil
		if !finished || err != nil || result != nil {
			if body != nil {
				err = errors.Join(err, body.Close())
			}
			return
		}
		req.Body = body
		req.ContentLength = body.size
		req.GetBody = func() (io.ReadCloser, error) { return body.reader(), nil }
	}()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if req.ContentLength < -1 {
		return nil, ErrContentLength
	}
	if req.ContentLength > budget.limits.BodyBytesMax {
		return nil, ErrBodyTooLarge
	}
	body, err = budget.acquire()
	if err != nil {
		return nil, err
	}
	if result := p.request.Validate(req); result != nil {
		return result, nil
	}
	contentType := req.Header.Get("Content-Type")
	mediaType, _, mediaErr := mime.ParseMediaType(contentType)
	if contentType != "" && mediaErr != nil {
		return &Result{Detail: "Invalid Content-Type", Errors: []ValidationError{{Message: "Content-Type must be a valid media type with valid parameters", Location: "header.Content-Type", Fix: nil}}}, nil
	}
	stream := p.streaming(req, mediaType)
	limit := budget.limits.BodyBytesMax
	if stream == nil {
		limit = budget.limits.BufferedBodyBytesMax
	}
	if req.ContentLength > limit {
		return nil, ErrBodyTooLarge
	}
	lengthKnown := req.ContentLength >= 0 && (req.ContentLength > 0 || input == http.NoBody || req.RequestURI != "")
	if lengthKnown {
		limit = req.ContentLength
	}
	reader := &capturingReader{ctx: ctx, input: input, body: body, limit: limit, lengthKnown: lengthKnown, eof: false, failure: nil, emptyReads: 0}
	check := stream
	if check == nil && strings.Contains(strings.ToLower(contentType), "json") {
		check = p.syntax
	}
	if check != nil {
		var first [1]byte
		n, firstErr := io.ReadFull(reader, first[:])
		switch {
		case firstErr == io.EOF:
			result, err = p.buffered(req, body)
		case firstErr != nil:
			err = firstErr
		default:
			validateErr := check.Validate(ctx, io.MultiReader(bytes.NewReader(first[:n]), reader))
			var invalid *streamschema.ValidationError
			if errors.As(validateErr, &invalid) {
				if invalid.Reason == streamschema.ReasonLimit {
					err = validateErr
				} else {
					result = &Result{Detail: "Request body failed validation", Errors: []ValidationError{{Message: invalid.Message, Location: invalid.InstanceLocation, Fix: nil}}}
				}
			} else {
				err = validateErr
			}
			if err == nil && result == nil && stream == nil {
				result, err = p.buffered(req, body)
			}
		}
	} else {
		_, err = io.Copy(io.Discard, reader)
		if err == nil {
			result, err = p.buffered(req, body)
		}
	}
	if reader.failure != nil {
		return nil, reader.failure
	}
	if err != nil {
		return nil, err
	}
	finished = true
	return result, nil
}

func (p *Plan) streaming(req *http.Request, mediaType string) *streamschema.Schema {
	if mediaType != "application/json" {
		return nil
	}
	route, err := p.routes.FindRoute(req)
	if err != nil || route.Operation == nil || route.Operation.RequestBody == nil {
		return nil
	}
	return p.streams[route.Operation.RequestBody.Content.GetOrZero(mediaType)]
}

func (p *Plan) buffered(req *http.Request, body *retainedBody) (*Result, error) {
	budget := body.budget
	budget.mu.Lock()
	if body.size > (budget.limits.BytesMax-budget.bytes)/bufferedCopyFactor {
		budget.mu.Unlock()
		return nil, ErrCapacity
	}
	reserved := body.size * bufferedCopyFactor
	budget.bytes += reserved
	budget.mu.Unlock()
	defer func() {
		budget.mu.Lock()
		budget.bytes -= reserved
		budget.mu.Unlock()
	}()

	clone := req.Clone(req.Context())
	clone.Body = body.reader()
	clone.ContentLength = body.size
	clone.GetBody = nil
	return p.full.Validate(clone), nil
}

type capturingReader struct {
	ctx         context.Context
	input       io.Reader
	body        *retainedBody
	limit       int64
	lengthKnown bool
	eof         bool
	failure     error
	emptyReads  int
}

func (r *capturingReader) Read(p []byte) (int, error) {
	if r.failure != nil {
		return 0, r.failure
	}
	if len(p) == 0 {
		return 0, nil
	}
	if r.eof {
		return 0, io.EOF
	}
	if err := r.ctx.Err(); err != nil {
		r.failure = err
		return 0, err
	}
	if room := r.limit - r.body.size; int64(len(p)) > room {
		p = p[:room+1]
	}
	n, err := r.input.Read(p)
	switch {
	case n < 0 || n > len(p):
		r.failure = fault.New("input returned invalid byte count")
	case int64(n) > r.limit-r.body.size:
		r.failure = ErrBodyTooLarge
		if r.lengthKnown {
			r.failure = ErrContentLength
		}
	case n > 0:
		_, r.failure = r.body.Write(p[:n])
		r.emptyReads = 0
	case err == nil:
		r.emptyReads++
		if r.emptyReads >= 100 {
			r.failure = io.ErrNoProgress
		}
	}
	if err == io.EOF {
		r.eof = true
		if r.lengthKnown && r.body.size != r.limit {
			r.failure = errors.Join(r.failure, ErrContentLength)
		}
	} else if err != nil {
		r.failure = errors.Join(r.failure, err)
	}
	if r.failure != nil {
		return 0, r.failure
	}
	return n, err
}
