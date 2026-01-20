package zen

import (
	"bufio"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// mockResponseWriter implements http.ResponseWriter with optional Flusher, Hijacker, and Pusher support.
type mockResponseWriter struct {
	http.ResponseWriter
	flushed       bool
	hijacked      bool
	pushed        bool
	pushTarget    string
	supportFlush  bool
	supportHijack bool
	supportPush   bool
}

func (m *mockResponseWriter) Flush() {
	m.flushed = true
}

func (m *mockResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	m.hijacked = true
	return nil, nil, nil
}

func (m *mockResponseWriter) Push(target string, opts *http.PushOptions) error {
	m.pushed = true
	m.pushTarget = target
	return nil
}

// fullMockWriter supports all optional interfaces
type fullMockWriter struct {
	*httptest.ResponseRecorder
	flushed    bool
	hijacked   bool
	pushed     bool
	pushTarget string
}

func newFullMockWriter() *fullMockWriter {
	return &fullMockWriter{
		ResponseRecorder: httptest.NewRecorder(),
	}
}

func (m *fullMockWriter) Flush() {
	m.flushed = true
	m.ResponseRecorder.Flush()
}

func (m *fullMockWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	m.hijacked = true
	return nil, nil, nil
}

func (m *fullMockWriter) Push(target string, opts *http.PushOptions) error {
	m.pushed = true
	m.pushTarget = target
	return nil
}

// basicMockWriter only implements http.ResponseWriter (no optional interfaces)
type basicMockWriter struct {
	*httptest.ResponseRecorder
}

func newBasicMockWriter() *basicMockWriter {
	return &basicMockWriter{
		ResponseRecorder: httptest.NewRecorder(),
	}
}

// --- ErrorCapturingWriter Tests ---

func TestErrorCapturingWriter_FlushDelegatesToUnderlying(t *testing.T) {
	mock := newFullMockWriter()
	w := NewErrorCapturingWriter(mock)

	w.Flush()

	require.True(t, mock.flushed, "Flush should delegate to underlying writer")
}

func TestErrorCapturingWriter_FlushNoOpWhenErrorCaptured(t *testing.T) {
	mock := newFullMockWriter()
	w := NewErrorCapturingWriter(mock)
	w.SetError(errors.New("test error"))

	w.Flush()

	require.False(t, mock.flushed, "Flush should be no-op when error is captured")
}

func TestErrorCapturingWriter_FlushWritesHeadersFirst(t *testing.T) {
	mock := newFullMockWriter()
	w := NewErrorCapturingWriter(mock)

	w.Flush()

	require.Equal(t, http.StatusOK, mock.Code, "Flush should write 200 OK header")
}

func TestErrorCapturingWriter_HijackDelegatesToUnderlying(t *testing.T) {
	mock := newFullMockWriter()
	w := NewErrorCapturingWriter(mock)

	_, _, err := w.Hijack()

	require.NoError(t, err, "Hijack should succeed")
	require.True(t, mock.hijacked, "Hijack should delegate to underlying writer")
}

func TestErrorCapturingWriter_HijackReturnsErrorWhenErrorCaptured(t *testing.T) {
	mock := newFullMockWriter()
	w := NewErrorCapturingWriter(mock)
	w.SetError(errors.New("test error"))

	_, _, err := w.Hijack()

	require.ErrorIs(t, err, ErrHijackAfterError, "Hijack should return ErrHijackAfterError")
	require.False(t, mock.hijacked, "Hijack should not delegate when error is captured")
}

func TestErrorCapturingWriter_HijackReturnsErrorWhenNotSupported(t *testing.T) {
	mock := newBasicMockWriter()
	w := NewErrorCapturingWriter(mock)

	_, _, err := w.Hijack()

	require.ErrorIs(t, err, ErrHijackNotSupported, "Hijack should return ErrHijackNotSupported")
}

func TestErrorCapturingWriter_PushDelegatesToUnderlying(t *testing.T) {
	mock := newFullMockWriter()
	w := NewErrorCapturingWriter(mock)

	err := w.Push("/test", nil)

	require.NoError(t, err, "Push should succeed")
	require.True(t, mock.pushed, "Push should delegate to underlying writer")
	require.Equal(t, "/test", mock.pushTarget, "Push target should be /test")
}

func TestErrorCapturingWriter_PushReturnsErrorWhenErrorCaptured(t *testing.T) {
	mock := newFullMockWriter()
	w := NewErrorCapturingWriter(mock)
	w.SetError(errors.New("test error"))

	err := w.Push("/test", nil)

	require.ErrorIs(t, err, ErrPushNotSupported, "Push should return ErrPushNotSupported when error captured")
	require.False(t, mock.pushed, "Push should not delegate when error is captured")
}

func TestErrorCapturingWriter_PushReturnsErrorWhenNotSupported(t *testing.T) {
	mock := newBasicMockWriter()
	w := NewErrorCapturingWriter(mock)

	err := w.Push("/test", nil)

	require.ErrorIs(t, err, ErrPushNotSupported, "Push should return ErrPushNotSupported")
}

func TestErrorCapturingWriter_WriteDiscardsWhenErrorCaptured(t *testing.T) {
	mock := newFullMockWriter()
	w := NewErrorCapturingWriter(mock)
	w.SetError(errors.New("test error"))

	n, err := w.Write([]byte("test"))

	require.NoError(t, err, "Write should not return error")
	require.Equal(t, 4, n, "Write should return byte count")
	require.Zero(t, mock.Body.Len(), "Write should discard body when error is captured")
}

func TestErrorCapturingWriter_TypeAssertions(t *testing.T) {
	mock := newFullMockWriter()
	w := NewErrorCapturingWriter(mock)

	_, ok := interface{}(w).(http.Flusher)
	require.True(t, ok, "ErrorCapturingWriter should implement http.Flusher")
	_, ok = interface{}(w).(http.Hijacker)
	require.True(t, ok, "ErrorCapturingWriter should implement http.Hijacker")
	_, ok = interface{}(w).(http.Pusher)
	require.True(t, ok, "ErrorCapturingWriter should implement http.Pusher")
}

// --- statusRecorder Tests ---

func TestStatusRecorder_FlushDelegatesToUnderlying(t *testing.T) {
	mock := newFullMockWriter()
	r := &statusRecorder{ResponseWriter: mock}

	r.Flush()

	require.True(t, mock.flushed, "Flush should delegate to underlying writer")
}

func TestStatusRecorder_FlushSetsWrittenAndStatusCode(t *testing.T) {
	mock := newFullMockWriter()
	r := &statusRecorder{ResponseWriter: mock}

	r.Flush()

	require.True(t, r.written, "Flush should set written = true")
	require.Equal(t, http.StatusOK, r.statusCode, "Flush should set statusCode to 200")
}

func TestStatusRecorder_HijackDelegatesToUnderlying(t *testing.T) {
	mock := newFullMockWriter()
	r := &statusRecorder{ResponseWriter: mock}

	_, _, err := r.Hijack()

	require.NoError(t, err, "Hijack should succeed")
	require.True(t, mock.hijacked, "Hijack should delegate to underlying writer")
}

func TestStatusRecorder_HijackReturnsErrorWhenNotSupported(t *testing.T) {
	mock := newBasicMockWriter()
	r := &statusRecorder{ResponseWriter: mock}

	_, _, err := r.Hijack()

	require.ErrorIs(t, err, ErrHijackNotSupported, "Hijack should return ErrHijackNotSupported")
}

func TestStatusRecorder_PushDelegatesToUnderlying(t *testing.T) {
	mock := newFullMockWriter()
	r := &statusRecorder{ResponseWriter: mock}

	err := r.Push("/test", nil)

	require.NoError(t, err, "Push should succeed")
	require.True(t, mock.pushed, "Push should delegate to underlying writer")
}

func TestStatusRecorder_PushReturnsErrorWhenNotSupported(t *testing.T) {
	mock := newBasicMockWriter()
	r := &statusRecorder{ResponseWriter: mock}

	err := r.Push("/test", nil)

	require.ErrorIs(t, err, ErrPushNotSupported, "Push should return ErrPushNotSupported")
}

func TestStatusRecorder_TypeAssertions(t *testing.T) {
	mock := newFullMockWriter()
	r := &statusRecorder{ResponseWriter: mock}

	_, ok := interface{}(r).(http.Flusher)
	require.True(t, ok, "statusRecorder should implement http.Flusher")
	_, ok = interface{}(r).(http.Hijacker)
	require.True(t, ok, "statusRecorder should implement http.Hijacker")
	_, ok = interface{}(r).(http.Pusher)
	require.True(t, ok, "statusRecorder should implement http.Pusher")
}
