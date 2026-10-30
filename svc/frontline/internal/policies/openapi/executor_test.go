package openapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"os"
	"testing"
	"testing/iotest"
	"time"

	"github.com/stretchr/testify/require"
	frontlinev1 "github.com/unkeyed/unkey/gen/proto/frontline/v1"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/zen"
)

var minimalSpec = []byte(`
openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths:
  /users:
    post:
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [name]
              properties:
                name:
                  type: string
      responses:
        "200":
          description: ok
`)

const (
	testBodyBytesMax    = 2 << 20
	testBodyBytesBudget = decodeExpansionMax * testBodyBytesMax
)

func newTestExecutor(t *testing.T) *Executor {
	t.Helper()
	e, err := New(clock.New(), testBodyBytesMax, testBodyBytesBudget)
	require.NoError(t, err)
	t.Cleanup(e.Close)
	return e
}

func newTestSession(t *testing.T) *zen.Session {
	t.Helper()
	sess := &zen.Session{}
	require.NoError(t, sess.Init(&deadlineWriter{ResponseRecorder: httptest.NewRecorder()}, httptest.NewRequest("GET", "/", nil), 0))
	return sess
}

type deadlineWriter struct {
	*httptest.ResponseRecorder
	deadlines []time.Time
}

func (w *deadlineWriter) SetReadDeadline(deadline time.Time) error {
	w.deadlines = append(w.deadlines, deadline)
	return nil
}

func jsonBody(t *testing.T, value any) []byte {
	t.Helper()
	body, err := json.Marshal(value)
	require.NoError(t, err)
	return body
}

func TestExecute_EmptySpec(t *testing.T) {
	t.Parallel()

	e := newTestExecutor(t)
	req := httptest.NewRequest("GET", "/anything", nil)

	//nolint:exhaustruct
	redactor, err := e.Execute(context.Background(), newTestSession(t), req, &frontlinev1.OpenApiRequestValidation{})
	require.NoError(t, err)
	require.Nil(t, redactor)
}

func TestExecute_ValidRequest(t *testing.T) {
	t.Parallel()

	e := newTestExecutor(t)
	body := jsonBody(t, map[string]string{"name": "alice"})
	req := httptest.NewRequest("POST", "/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	_, err := e.Execute(context.Background(), newTestSession(t), req, &frontlinev1.OpenApiRequestValidation{
		SpecYaml: minimalSpec,
	})
	require.NoError(t, err)

	forwardedBody, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	require.Equal(t, body, forwardedBody)
}

func TestExecute_InvalidRequest(t *testing.T) {
	t.Parallel()

	e := newTestExecutor(t)
	body := jsonBody(t, map[string]string{"wrong": "field"})
	req := httptest.NewRequest("POST", "/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	_, err := e.Execute(context.Background(), newTestSession(t), req, &frontlinev1.OpenApiRequestValidation{
		SpecYaml: minimalSpec,
	})
	require.Error(t, err)

	urn, ok := fault.GetCode(err)
	require.True(t, ok)
	require.Equal(t, codes.Frontline.OpenApi.InvalidRequest.URN(), urn)
}

func TestExecute_BodyExceedsLimit(t *testing.T) {
	t.Parallel()

	body := []byte(`{"name":"` + string(bytes.Repeat([]byte("K"), testBodyBytesMax)) + `"}`)

	tests := []struct {
		name          string
		contentLength int64
	}{
		{name: "declared length", contentLength: int64(len(body))},
		{name: "unknown length", contentLength: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest("POST", "/users", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.ContentLength = tt.contentLength

			_, err := newTestExecutor(t).Execute(context.Background(), newTestSession(t), req, &frontlinev1.OpenApiRequestValidation{
				SpecYaml: minimalSpec,
			})
			require.Error(t, err)

			urn, ok := fault.GetCode(err)
			require.True(t, ok)
			require.Equal(t, codes.User.BadRequest.RequestBodyTooLarge.URN(), urn)
		})
	}
}

func TestExecute_BodyExceedsLimitDoesNotReadBody(t *testing.T) {
	t.Parallel()

	e := newTestExecutor(t)
	reader := &countingReader{data: bytes.Repeat([]byte("K"), int(testBodyBytesMax)*4)}
	reader.remaining = len(reader.data)
	req := httptest.NewRequest("POST", "/users", io.NopCloser(reader))
	req.Header.Set("Content-Type", "application/json")
	req.ContentLength = int64(reader.remaining)

	_, err := e.Execute(context.Background(), newTestSession(t), req, &frontlinev1.OpenApiRequestValidation{
		SpecYaml: minimalSpec,
	})
	require.Error(t, err)
	require.Zero(t, reader.reads)
}

func TestExecute_BodyAtLimitPasses(t *testing.T) {
	t.Parallel()

	e := newTestExecutor(t)
	pad := testBodyBytesMax - len(jsonBody(t, map[string]string{"name": ""}))
	body := jsonBody(t, map[string]string{"name": string(bytes.Repeat([]byte("K"), pad))})
	require.Len(t, body, testBodyBytesMax)
	req := httptest.NewRequest("POST", "/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.ContentLength = -1

	_, err := e.Execute(context.Background(), newTestSession(t), req, &frontlinev1.OpenApiRequestValidation{
		SpecYaml: minimalSpec,
	})
	require.NoError(t, err)

	forwarded, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	require.Equal(t, body, forwarded)
	require.NotNil(t, req.GetBody)

	replayed, err := req.GetBody()
	require.NoError(t, err)
	replayedBody, err := io.ReadAll(replayed)
	require.NoError(t, err)
	require.Equal(t, body, replayedBody)
}

func TestExecute_LimitDisabled(t *testing.T) {
	t.Parallel()

	e, err := New(clock.New(), 0, 0)
	require.NoError(t, err)
	t.Cleanup(e.Close)

	body := []byte(`{"name":"` + string(bytes.Repeat([]byte("K"), testBodyBytesMax)) + `"}`)
	req := httptest.NewRequest("POST", "/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.ContentLength = -1

	_, err = e.Execute(context.Background(), newTestSession(t), req, &frontlinev1.OpenApiRequestValidation{
		SpecYaml: minimalSpec,
	})
	require.NoError(t, err)
}

func TestLimitBody_BodyIsBuffered(t *testing.T) {
	t.Parallel()

	body := jsonBody(t, map[string]string{"name": "alice"})
	for _, contentLength := range []int64{int64(len(body)), -1} {
		e := newTestExecutor(t)
		reader := &countingReader{remaining: len(body), data: body}
		req := httptest.NewRequest("POST", "/users", io.NopCloser(reader))
		req.ContentLength = contentLength

		require.NoError(t, e.limitBody(newTestSession(t).ResponseWriter(), req))
		require.Zero(t, reader.remaining)
		require.NotNil(t, req.GetBody)
		require.EqualValues(t, len(body), req.ContentLength)

		forwarded, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		require.Equal(t, body, forwarded)
	}
}

func TestLimitBody_BudgetExhausted(t *testing.T) {
	t.Parallel()

	e := newTestExecutor(t)
	body := []byte("KEBAP")
	require.True(t, e.bodyBudget.TryAcquire(testBodyBytesBudget-int64(len(body))))

	holderCtx, endHolder := context.WithCancel(t.Context())
	t.Cleanup(endHolder)
	holder := httptest.NewRequestWithContext(holderCtx, "POST", "/users", io.NopCloser(&countingReader{data: body, remaining: len(body)}))
	holder.ContentLength = int64(len(body))
	require.NoError(t, e.limitBody(newTestSession(t).ResponseWriter(), holder))

	reader := &countingReader{data: body, remaining: len(body)}
	req := httptest.NewRequestWithContext(t.Context(), "POST", "/users", io.NopCloser(reader))
	req.ContentLength = int64(len(body))

	err := e.limitBody(newTestSession(t).ResponseWriter(), req)
	require.Error(t, err)
	urn, ok := fault.GetCode(err)
	require.True(t, ok)
	require.Equal(t, codes.Frontline.OpenApi.ValidationBusy.URN(), urn)
	require.Zero(t, reader.reads)

	endHolder()
	require.Eventually(t, func() bool {
		return e.limitBody(newTestSession(t).ResponseWriter(), req) == nil
	}, time.Second, 5*time.Millisecond)
}

func TestLimitBody_ChunkedBodyHoldsOnlyItsSize(t *testing.T) {
	t.Parallel()

	e := newTestExecutor(t)

	body := []byte("KEBAP")
	for range 2 * decodeExpansionMax {
		req := httptest.NewRequestWithContext(t.Context(), "POST", "/users", io.NopCloser(&countingReader{data: body, remaining: len(body)}))
		req.ContentLength = -1
		require.NoError(t, e.limitBody(newTestSession(t).ResponseWriter(), req))
	}
}

func TestLimitBody_FailedChunkedBodyReleasesBudget(t *testing.T) {
	t.Parallel()

	e := newTestExecutor(t)

	body := bytes.Repeat([]byte("K"), testBodyBytesMax+1)
	tooLarge := httptest.NewRequestWithContext(t.Context(), "POST", "/users", io.NopCloser(&countingReader{data: body, remaining: len(body)}))
	tooLarge.ContentLength = -1
	err := e.limitBody(newTestSession(t).ResponseWriter(), tooLarge)
	urn, ok := fault.GetCode(err)
	require.True(t, ok)
	require.Equal(t, codes.User.BadRequest.RequestBodyTooLarge.URN(), urn)

	require.True(t, e.bodyBudget.TryAcquire(testBodyBytesBudget))
}

func TestExecute_DecodeBudgetExhausted(t *testing.T) {
	t.Parallel()

	e := newTestExecutor(t)
	body := jsonBody(t, map[string]string{"name": "KEBAP"})
	bytesFree := decodeExpansionMax*int64(len(body)) - 1
	require.True(t, e.bodyBudget.TryAcquire(testBodyBytesBudget-bytesFree))

	busyCtx, endBusy := context.WithCancel(t.Context())
	t.Cleanup(endBusy)
	req := httptest.NewRequestWithContext(busyCtx, "POST", "/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	_, err := e.Execute(busyCtx, newTestSession(t), req, &frontlinev1.OpenApiRequestValidation{SpecYaml: minimalSpec})
	urn, ok := fault.GetCode(err)
	require.True(t, ok)
	require.Equal(t, codes.Frontline.OpenApi.ValidationBusy.URN(), urn)

	endBusy()
	e.bodyBudget.Release(1)
	require.Eventually(t, func() bool {
		if !e.bodyBudget.TryAcquire(bytesFree + 1) {
			return false
		}
		e.bodyBudget.Release(bytesFree + 1)
		return true
	}, time.Second, 5*time.Millisecond)

	req = httptest.NewRequestWithContext(t.Context(), "POST", "/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	_, err = e.Execute(t.Context(), newTestSession(t), req, &frontlinev1.OpenApiRequestValidation{SpecYaml: minimalSpec})
	require.NoError(t, err)
	require.True(t, e.bodyBudget.TryAcquire(bytesFree+1-int64(len(body))))
}

func TestLimitBody_ClearsReadDeadline(t *testing.T) {
	t.Parallel()

	e := newTestExecutor(t)
	body := []byte("KEBAP")
	w := &deadlineWriter{ResponseRecorder: httptest.NewRecorder()}
	req := httptest.NewRequestWithContext(t.Context(), "POST", "/users", bytes.NewReader(body))

	require.NoError(t, e.limitBody(w, req))
	require.Len(t, w.deadlines, 2)
	require.False(t, w.deadlines[0].IsZero())
	require.True(t, w.deadlines[1].IsZero())
}

func TestLimitBody_SlowBodySaysWhy(t *testing.T) {
	t.Parallel()

	e := newTestExecutor(t)
	req := httptest.NewRequestWithContext(t.Context(), "POST", "/users", io.NopCloser(iotest.ErrReader(os.ErrDeadlineExceeded)))
	req.ContentLength = 5

	err := e.limitBody(newTestSession(t).ResponseWriter(), req)
	urn, ok := fault.GetCode(err)
	require.True(t, ok)
	require.Equal(t, codes.User.BadRequest.RequestBodyUnreadable.URN(), urn)
	require.Equal(t, "The request body did not arrive within 30s.", fault.UserFacingMessage(err))
}

func TestExecute_TwoPoliciesReserveBodyOnce(t *testing.T) {
	t.Parallel()

	e := newTestExecutor(t)
	body := jsonBody(t, map[string]string{"name": "KEBAP"})
	req := httptest.NewRequestWithContext(t.Context(), "POST", "/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	cfg := &frontlinev1.OpenApiRequestValidation{SpecYaml: minimalSpec}

	for range 2 {
		_, err := e.Execute(t.Context(), newTestSession(t), req, cfg)
		require.NoError(t, err)
	}
	require.True(t, e.bodyBudget.TryAcquire(testBodyBytesBudget-int64(len(body))))
}

func TestNew_BudgetSmallerThanLimit(t *testing.T) {
	t.Parallel()

	_, err := New(clock.New(), testBodyBytesMax, testBodyBytesBudget-1)
	require.Error(t, err)
}

type countingReader struct {
	data      []byte
	remaining int
	reads     int
}

func (r *countingReader) Read(p []byte) (int, error) {
	r.reads++
	if r.remaining == 0 {
		return 0, io.EOF
	}
	n := min(len(p), r.remaining)
	start := len(r.data) - r.remaining
	copy(p, r.data[start:start+n])
	r.remaining -= n
	return n, nil
}

func TestExecute_InvalidSpec(t *testing.T) {
	t.Parallel()

	e := newTestExecutor(t)
	req := httptest.NewRequest("GET", "/anything", nil)

	_, err := e.Execute(context.Background(), newTestSession(t), req, &frontlinev1.OpenApiRequestValidation{
		SpecYaml: []byte("not valid yaml: [[["),
	})
	require.Error(t, err)

	urn, ok := fault.GetCode(err)
	require.True(t, ok)
	require.Equal(t, codes.Frontline.Internal.InvalidConfiguration.URN(), urn)
}

func TestExecute_CachesCompiledValidator(t *testing.T) {
	t.Parallel()

	e := newTestExecutor(t)
	cfg := &frontlinev1.OpenApiRequestValidation{SpecYaml: minimalSpec}

	body := jsonBody(t, map[string]string{"name": "alice"})
	req1 := httptest.NewRequest("POST", "/users", bytes.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	_, err := e.Execute(context.Background(), newTestSession(t), req1, cfg)
	require.NoError(t, err)

	req2 := httptest.NewRequest("POST", "/users", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	_, err = e.Execute(context.Background(), newTestSession(t), req2, cfg)
	require.NoError(t, err)

	_, hit := e.cache.Get(context.Background(), string(minimalSpec))
	require.Equal(t, cache.Hit, hit)
}

func TestExecute_ReturnsSpecBodyRedactor(t *testing.T) {
	t.Parallel()

	spec := []byte(`
openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths:
  /secrets:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                secret:
                  type: string
                  x-unkey-redact: true
                visible:
                  type: string
                  x-unkey-redact: false
      responses:
        "200":
          description: ok
`)
	body := jsonBody(t, map[string]string{"secret": "hide-me", "visible": "keep-me"})
	req := httptest.NewRequest("POST", "/secrets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	redactor, err := newTestExecutor(t).Execute(context.Background(), newTestSession(t), req, &frontlinev1.OpenApiRequestValidation{
		SpecYaml: spec,
	})
	require.NoError(t, err)
	require.NotNil(t, redactor)
	require.Equal(t, `{"secret":"[REDACTED]","visible":"keep-me"}`, string(redactor.Redact(body)))
}
