package validation_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/openapi/validation"
)

func planJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	require.NoError(t, err)
	return data
}

func planSpec(t *testing.T, version string, schema any, required bool, mediaTypes ...string) []byte {
	t.Helper()
	content := map[string]any{"application/json": map[string]any{"schema": schema}}
	for _, mediaType := range mediaTypes {
		content[mediaType] = map[string]any{"schema": schema}
	}
	return planJSON(t, map[string]any{
		"openapi": version,
		"info":    map[string]any{"title": "Synthetic uploads", "version": "1"},
		"paths": map[string]any{"/upload": map[string]any{"post": map[string]any{
			"parameters":  []any{map[string]any{"name": "count", "in": "query", "required": true, "schema": map[string]any{"type": "integer", "minimum": 1}}},
			"requestBody": map[string]any{"required": required, "content": content},
			"responses":   map[string]any{"200": map[string]any{"description": "accepted"}},
		}}},
		"components": map[string]any{"schemas": map[string]any{"Text": map[string]any{"type": "string"}}},
	})
}

func planRequest(t *testing.T, payload []byte) *http.Request {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "http://example.test/upload?count=1", bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	t.Cleanup(func() { require.NoError(t, req.Body.Close()) })
	return req
}

func planBudget(t *testing.T) *validation.Budget {
	t.Helper()
	b, err := validation.NewBudget(validation.Limits{BodyBytesMax: 32 << 20, BufferedBodyBytesMax: 1 << 20, BytesMax: 64 << 20, RequestsMax: 8})
	require.NoError(t, err)
	return b
}

func TestPlanSchemaCoverage(t *testing.T) {
	for _, test := range []struct {
		name    string
		version string
		schema  any
		valid   any
		invalid any
	}{
		{"type", "3.1.0", map[string]any{"type": "integer"}, 2, "two"},
		{"nullable", "3.0.3", map[string]any{"type": "string", "nullable": true}, nil, 2},
		{"enum", "3.1.0", map[string]any{"enum": []string{"red", "blue"}}, "red", "green"},
		{"const", "3.1.0", map[string]any{"const": 3}, 3, 4},
		{"pattern", "3.1.0", map[string]any{"type": "string", "pattern": "^[a-z]+$"}, "abc", "12"},
		{"multipleOf", "3.1.0", map[string]any{"type": "integer", "multipleOf": 3}, 6, 7},
		{"oneOf", "3.1.0", map[string]any{"oneOf": []any{map[string]any{"type": "number"}, map[string]any{"type": "integer"}}}, 2.5, 2},
		{"allOf", "3.1.0", map[string]any{"allOf": []any{map[string]any{"minimum": 2}, map[string]any{"maximum": 4}}}, 3, 5},
		{"anyOf", "3.1.0", map[string]any{"anyOf": []any{map[string]any{"type": "boolean"}, map[string]any{"type": "string"}}}, false, 3},
		{"not", "3.1.0", map[string]any{"not": map[string]any{"type": "string"}}, 3, "three"},
		{"uniqueItems", "3.1.0", map[string]any{"type": "array", "uniqueItems": true}, []int{1, 2}, []int{1, 1}},
		{"contains", "3.1.0", map[string]any{"type": "array", "contains": map[string]any{"const": 3}, "minContains": 2}, []int{3, 3}, []int{3, 4}},
		{"prefixItems", "3.1.0", map[string]any{"type": "array", "prefixItems": []any{map[string]any{"type": "string"}}, "items": false}, []any{"a"}, []any{"a", 2}},
		{"dependentRequired", "3.1.0", map[string]any{"type": "object", "dependentRequired": map[string]any{"a": []string{"b"}}}, map[string]any{"a": 1, "b": 2}, map[string]any{"a": 1}},
		{"unevaluatedProperties", "3.1.0", map[string]any{"type": "object", "allOf": []any{map[string]any{"properties": map[string]any{"a": true}}}, "unevaluatedProperties": false}, map[string]any{"a": 1}, map[string]any{"b": 1}},
		{"conditional", "3.1.0", map[string]any{"if": map[string]any{"type": "integer"}, "then": map[string]any{"minimum": 5}, "else": map[string]any{"type": "string"}}, 6, 4},
		{"reference", "3.1.0", map[string]any{"$ref": "#/components/schemas/Text"}, "word", 3},
		{"readOnly required", "3.1.0", map[string]any{"type": "object", "properties": map[string]any{"server": map[string]any{"type": "string", "readOnly": true}, "client": map[string]any{"type": "string"}}, "required": []string{"server", "client"}}, map[string]any{"client": "a"}, map[string]any{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			spec := planSpec(t, test.version, test.schema, true)
			plan, err := validation.Compile(spec)
			require.NoError(t, err)
			baseline, err := validation.NewFromBytes(spec)
			require.NoError(t, err)
			budget := planBudget(t)
			for _, valid := range []bool{true, false} {
				value := test.invalid
				if valid {
					value = test.valid
				}
				payload := planJSON(t, value)
				require.Equal(t, valid, baseline.Validate(planRequest(t, payload)) == nil)
				req := planRequest(t, payload)
				result, err := plan.Validate(req, budget)
				require.NoError(t, err)
				require.Equal(t, valid, result == nil)
				if valid {
					replay, err := io.ReadAll(req.Body)
					require.NoError(t, err)
					require.Equal(t, payload, replay)
				}
				require.NoError(t, req.Body.Close())
			}
		})
	}
}

func TestPlanPreservesRequestChecks(t *testing.T) {
	plan, err := validation.Compile(planSpec(t, "3.1.0", map[string]any{"type": "string"}, true))
	require.NoError(t, err)
	for _, change := range []func(*http.Request){
		func(r *http.Request) { r.URL.RawQuery = "count=0" },
		func(r *http.Request) { r.URL.Path = "/absent" },
		func(r *http.Request) { r.Method = http.MethodDelete },
		func(r *http.Request) { r.Header.Del("Content-Type") },
		func(r *http.Request) { r.Header.Set("Content-Type", "text/plain") },
	} {
		req := planRequest(t, planJSON(t, "hello"))
		change(req)
		result, err := plan.Validate(req, planBudget(t))
		require.NoError(t, err)
		require.NotNil(t, result)
	}
	for _, required := range []bool{false, true} {
		plan, err := validation.Compile(planSpec(t, "3.1.0", map[string]any{"type": "string"}, required))
		require.NoError(t, err)
		result, err := plan.Validate(planRequest(t, nil), planBudget(t))
		require.NoError(t, err)
		require.Equal(t, required, result != nil)
	}
}

func TestPlanStreamsLargeBodiesAndBoundsFallback(t *testing.T) {
	payload := planJSON(t, strings.Repeat("x", 16<<20))
	for _, lengthKnown := range []bool{true, false} {
		plan, err := validation.Compile(planSpec(t, "3.1.0", map[string]any{"type": "string"}, true))
		require.NoError(t, err)
		req := planRequest(t, payload)
		if !lengthKnown {
			req.ContentLength = -1
		}
		result, err := plan.Validate(req, planBudget(t))
		require.NoError(t, err)
		require.Nil(t, result)
		replay, err := req.GetBody()
		require.NoError(t, err)
		got, err := io.ReadAll(replay)
		require.NoError(t, err)
		require.Equal(t, payload, got)
		require.NoError(t, replay.Close())
		require.NoError(t, req.Body.Close())
		_, err = replay.Read(make([]byte, 1))
		require.ErrorIs(t, err, io.ErrClosedPipe)
		fallback, err := validation.Compile(planSpec(t, "3.1.0", map[string]any{"type": "string", "pattern": "x"}, true))
		require.NoError(t, err)
		req = planRequest(t, payload)
		if !lengthKnown {
			req.ContentLength = -1
		}
		_, err = fallback.Validate(req, planBudget(t))
		require.ErrorIs(t, err, validation.ErrBodyTooLarge)
	}
}

func TestPlanSharedBudgetAndRecovery(t *testing.T) {
	first, err := validation.Compile(planSpec(t, "3.1.0", map[string]any{"type": "string"}, true))
	require.NoError(t, err)
	second, err := validation.Compile(planSpec(t, "3.1.0", map[string]any{"type": "string", "pattern": "a"}, true))
	require.NoError(t, err)
	for _, limits := range []validation.Limits{
		{BodyBytesMax: 1 << 20, BufferedBodyBytesMax: 1 << 20, BytesMax: 2 * (64<<10 + 1024), RequestsMax: 8},
		{BodyBytesMax: 1 << 20, BufferedBodyBytesMax: 1 << 20, BytesMax: 1 << 20, RequestsMax: 2},
	} {
		budget, err := validation.NewBudget(limits)
		require.NoError(t, err)
		a := planRequest(t, planJSON(t, "a"))
		a.ContentLength = -1
		result, err := first.Validate(a, budget)
		require.NoError(t, err)
		require.Nil(t, result)
		b := planRequest(t, planJSON(t, "a"))
		b.ContentLength = -1
		result, err = second.Validate(b, budget)
		require.NoError(t, err)
		require.Nil(t, result)
		_, err = first.Validate(planRequest(t, planJSON(t, "a")), budget)
		require.ErrorIs(t, err, validation.ErrCapacity)
		require.NoError(t, a.Body.Close())
		require.NoError(t, a.Body.Close())
		c := planRequest(t, planJSON(t, "a"))
		result, err = first.Validate(c, budget)
		require.NoError(t, err)
		require.Nil(t, result)
		require.NoError(t, b.Body.Close())
		require.NoError(t, c.Body.Close())
	}
}

type failingBody struct {
	payload  []byte
	readErr  error
	closeErr error
	closes   atomic.Int32
}

func (b *failingBody) Read(p []byte) (int, error) {
	n := copy(p, b.payload)
	b.payload = b.payload[n:]
	if len(b.payload) > 0 && b.readErr == io.EOF {
		return n, nil
	}
	return n, b.readErr
}

func (b *failingBody) Close() error { b.closes.Add(1); return b.closeErr }

func TestPlanFailureReleasesBudget(t *testing.T) {
	plan, err := validation.Compile(planSpec(t, "3.1.0", map[string]any{"type": "string"}, true))
	require.NoError(t, err)
	sourceErr := errors.New("synthetic source failure")
	for _, test := range []struct {
		name     string
		payload  []byte
		length   int64
		readErr  error
		closeErr error
		want     error
	}{
		{"source", planJSON(t, "a"), -1, sourceErr, nil, sourceErr},
		{"close", planJSON(t, "a"), -1, io.EOF, sourceErr, sourceErr},
		{"short declaration", planJSON(t, "a"), 2, io.EOF, nil, validation.ErrContentLength},
		{"long declaration", planJSON(t, "a"), 4, io.EOF, nil, validation.ErrContentLength},
		{"known oversize", planJSON(t, "a"), 1 << 30, io.EOF, nil, validation.ErrBodyTooLarge},
	} {
		t.Run(test.name, func(t *testing.T) {
			budget, err := validation.NewBudget(validation.Limits{BodyBytesMax: 1 << 20, BufferedBodyBytesMax: 1 << 20, BytesMax: 1 << 20, RequestsMax: 1})
			require.NoError(t, err)
			req := planRequest(t, nil)
			body := &failingBody{payload: test.payload, readErr: test.readErr, closeErr: test.closeErr, closes: atomic.Int32{}}
			req.Body = body
			req.ContentLength = test.length
			_, err = plan.Validate(req, budget)
			require.ErrorIs(t, err, test.want)
			require.Equal(t, int32(1), body.closes.Load())
			result, err := plan.Validate(planRequest(t, planJSON(t, "a")), budget)
			require.NoError(t, err)
			require.Nil(t, result)
		})
	}
}

type blockingBody struct {
	started chan struct{}
	closed  chan struct{}
	once    sync.Once
}

func (b *blockingBody) Read([]byte) (int, error) {
	close(b.started)
	<-b.closed
	return 0, io.ErrClosedPipe
}

func (b *blockingBody) Close() error { b.once.Do(func() { close(b.closed) }); return nil }

func TestPlanCancellationReleasesBudget(t *testing.T) {
	plan, err := validation.Compile(planSpec(t, "3.1.0", map[string]any{"type": "string"}, true))
	require.NoError(t, err)
	budget, err := validation.NewBudget(validation.Limits{BodyBytesMax: 1 << 20, BufferedBodyBytesMax: 1 << 20, BytesMax: 1 << 20, RequestsMax: 1})
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	req := planRequest(t, nil).WithContext(ctx)
	body := &blockingBody{started: make(chan struct{}), closed: make(chan struct{}), once: sync.Once{}}
	req.Body = body
	req.ContentLength = -1
	done := make(chan error, 1)
	go func() { _, err := plan.Validate(req, budget); done <- err }()
	<-body.started
	cancel()
	require.ErrorIs(t, <-done, context.Canceled)
	result, err := plan.Validate(planRequest(t, planJSON(t, "a")), budget)
	require.NoError(t, err)
	require.Nil(t, result)
}

func TestPlanRejectsBeforeReading(t *testing.T) {
	plan, err := validation.Compile(planSpec(t, "3.1.0", map[string]any{"type": "string"}, true))
	require.NoError(t, err)
	budget, err := validation.NewBudget(validation.Limits{BodyBytesMax: 1024, BufferedBodyBytesMax: 512, BytesMax: 1 << 20, RequestsMax: 1})
	require.NoError(t, err)
	payload := planJSON(t, "hello")
	for _, length := range []int64{1025, int64(len(payload))} {
		req := planRequest(t, payload)
		source := bytes.NewReader(payload)
		req.Body = io.NopCloser(source)
		req.ContentLength = length
		if length == int64(len(payload)) {
			req.URL.RawQuery = "count=0"
		}
		result, err := plan.Validate(req, budget)
		require.True(t, result != nil || err != nil)
		require.Equal(t, len(payload), source.Len())
	}
	held := planRequest(t, payload)
	result, err := plan.Validate(held, budget)
	require.NoError(t, err)
	require.Nil(t, result)
	req := planRequest(t, payload)
	source := bytes.NewReader(payload)
	req.Body = io.NopCloser(source)
	_, err = plan.Validate(req, budget)
	require.ErrorIs(t, err, validation.ErrCapacity)
	require.Equal(t, len(payload), source.Len())
}

func TestPlanRejectsMalformedJSONOnBothPaths(t *testing.T) {
	for _, schema := range []any{map[string]any{"type": "object"}, map[string]any{"type": "object", "minProperties": 1}} {
		plan, err := validation.Compile(planSpec(t, "3.1.0", schema, true, "text/json", "application/problem+json", "text/plain"))
		require.NoError(t, err)
		for _, contentType := range []string{"application/json", "application/json; charset", "text/json", "application/problem+json", "text/plain; profile=json"} {
			t.Run(contentType, func(t *testing.T) {
				for _, payload := range []string{`{"a":1,"a":2}`, `{"a":1} {"a":2}`, `{"a":"\ud800"}`, `{"a":`} {
					req := planRequest(t, []byte(payload))
					req.Header.Set("Content-Type", contentType)
					result, err := plan.Validate(req, planBudget(t))
					require.NoError(t, err)
					require.NotNil(t, result)
				}
				req := planRequest(t, planJSON(t, map[string]int{"a": 1}))
				req.Header.Set("Content-Type", contentType)
				result, err := plan.Validate(req, planBudget(t))
				require.NoError(t, err)
				require.Equal(t, contentType == "application/json; charset", result != nil)
			})
		}
	}
}

func TestPlanClientUnknownLength(t *testing.T) {
	plan, err := validation.Compile(planSpec(t, "3.1.0", map[string]any{"type": "string"}, true))
	require.NoError(t, err)
	payload := planJSON(t, "hello")
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "http://example.test/upload?count=1", io.NopCloser(bytes.NewReader(payload)))
	require.NoError(t, err)
	require.Zero(t, req.ContentLength)
	req.Header.Set("Content-Type", "application/json")
	result, err := plan.Validate(req, planBudget(t))
	require.NoError(t, err)
	require.Nil(t, result)
	got, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	require.Equal(t, payload, got)
	require.NoError(t, req.Body.Close())

	req = planRequest(t, payload)
	req.ContentLength = 0
	req.RequestURI = "/upload?count=1"
	_, err = plan.Validate(req, planBudget(t))
	require.ErrorIs(t, err, validation.ErrContentLength)
}

func TestPlanFallbackReservesCopyCapacity(t *testing.T) {
	plan, err := validation.Compile(planSpec(t, "3.1.0", map[string]any{"type": "string", "pattern": "^x+$"}, true))
	require.NoError(t, err)
	payload := planJSON(t, strings.Repeat("x", 1<<20))
	for _, bytesMax := range []int64{2 << 20, 16 << 20} {
		budget, err := validation.NewBudget(validation.Limits{BodyBytesMax: 2 << 20, BufferedBodyBytesMax: 2 << 20, BytesMax: bytesMax, RequestsMax: 1})
		require.NoError(t, err)
		req := planRequest(t, payload)
		result, err := plan.Validate(req, budget)
		if bytesMax == 2<<20 {
			require.ErrorIs(t, err, validation.ErrCapacity)
		} else {
			require.NoError(t, err)
			require.Nil(t, result)
			got, err := io.ReadAll(req.Body)
			require.NoError(t, err)
			require.Equal(t, payload, got)
		}
		require.NoError(t, req.Body.Close())
		result, err = plan.Validate(planRequest(t, planJSON(t, "x")), budget)
		require.NoError(t, err)
		require.Nil(t, result)
	}
}

func TestPlanHTTPAdmission(t *testing.T) {
	for name, schema := range map[string]any{
		"streaming": map[string]any{"type": "string"},
		"fallback":  map[string]any{"type": "string", "pattern": "^x+$"},
	} {
		for _, protocol := range []string{"http1", "http2"} {
			for _, lengthKnown := range []bool{true, false} {
				t.Run(name+"/"+protocol+"/"+map[bool]string{true: "known", false: "unknown"}[lengthKnown], func(t *testing.T) {
					plan, err := validation.Compile(planSpec(t, "3.1.0", schema, true))
					require.NoError(t, err)
					budget, err := validation.NewBudget(validation.Limits{BodyBytesMax: 2 << 20, BufferedBodyBytesMax: 2 << 20, BytesMax: 32 << 20, RequestsMax: 2})
					require.NoError(t, err)
					payload := planJSON(t, strings.Repeat("x", 1<<20))
					admitted := make(chan struct{}, 2)
					release := make(chan struct{})
					var releaseOnce sync.Once
					unblock := func() { releaseOnce.Do(func() { close(release) }) }
					server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						result, err := plan.Validate(r, budget)
						if errors.Is(err, validation.ErrCapacity) {
							w.WriteHeader(http.StatusServiceUnavailable)
							return
						}
						if err != nil || result != nil {
							w.WriteHeader(http.StatusBadRequest)
							return
						}
						admitted <- struct{}{}
						<-release
						got, readErr := io.ReadAll(r.Body)
						closeErr := r.Body.Close()
						if readErr != nil || closeErr != nil || !bytes.Equal(payload, got) {
							w.WriteHeader(http.StatusInternalServerError)
							return
						}
						w.WriteHeader(http.StatusNoContent)
					}))
					server.EnableHTTP2 = protocol == "http2"
					server.StartTLS()
					t.Cleanup(server.Close)
					t.Cleanup(unblock)
					type response struct {
						value *http.Response
						err   error
					}
					responses := make(chan response, 3)
					request := func() *http.Request {
						t.Helper()
						r, err := http.NewRequestWithContext(t.Context(), http.MethodPost, server.URL+"/upload?count=1", bytes.NewReader(payload))
						require.NoError(t, err)
						r.Header.Set("Content-Type", "application/json")
						if !lengthKnown {
							r.ContentLength = -1
						}
						return r
					}
					for range 2 {
						r := request()
						go func() { value, err := server.Client().Do(r); responses <- response{value: value, err: err} }()
					}
					for range 2 {
						select {
						case <-admitted:
						case result := <-responses:
							if result.value != nil {
								require.NoError(t, result.value.Body.Close())
							}
							t.Fatalf("request failed before admission: %v, %v", result.value, result.err)
						case <-t.Context().Done():
							t.Fatal("admission canceled")
						}
					}
					busy, err := server.Client().Do(request())
					require.NoError(t, err)
					require.Equal(t, http.StatusServiceUnavailable, busy.StatusCode)
					require.NoError(t, busy.Body.Close())
					unblock()
					for range 2 {
						result := <-responses
						require.NoError(t, result.err)
						require.Equal(t, http.StatusNoContent, result.value.StatusCode)
						require.Equal(t, map[string]int{"http1": 1, "http2": 2}[protocol], result.value.ProtoMajor)
						require.NoError(t, result.value.Body.Close())
					}
					recovered, err := server.Client().Do(request())
					require.NoError(t, err)
					require.Equal(t, http.StatusNoContent, recovered.StatusCode)
					require.NoError(t, recovered.Body.Close())
				})
			}
		}
	}
}
