package zen

import (
	"bufio"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockResponseWriter implements http.ResponseWriter with optional Flusher, Hijacker, and Pusher support.
type mockResponseWriter struct {
	http.ResponseWriter
	flushed      bool
	hijacked     bool
	pushed       bool
	pushTarget   string
	supportFlush bool
	supportHijack bool
	supportPush  bool
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

	if !mock.flushed {
		t.Error("Flush should delegate to underlying writer")
	}
}

func TestErrorCapturingWriter_FlushNoOpWhenErrorCaptured(t *testing.T) {
	mock := newFullMockWriter()
	w := NewErrorCapturingWriter(mock)
	w.SetError(errors.New("test error"))

	w.Flush()

	if mock.flushed {
		t.Error("Flush should be no-op when error is captured")
	}
}

func TestErrorCapturingWriter_FlushWritesHeadersFirst(t *testing.T) {
	mock := newFullMockWriter()
	w := NewErrorCapturingWriter(mock)

	w.Flush()

	if mock.Code != http.StatusOK {
		t.Errorf("Flush should write 200 OK header, got %d", mock.Code)
	}
}

func TestErrorCapturingWriter_HijackDelegatesToUnderlying(t *testing.T) {
	mock := newFullMockWriter()
	w := NewErrorCapturingWriter(mock)

	_, _, err := w.Hijack()

	if err != nil {
		t.Errorf("Hijack should succeed, got error: %v", err)
	}
	if !mock.hijacked {
		t.Error("Hijack should delegate to underlying writer")
	}
}

func TestErrorCapturingWriter_HijackReturnsErrorWhenErrorCaptured(t *testing.T) {
	mock := newFullMockWriter()
	w := NewErrorCapturingWriter(mock)
	w.SetError(errors.New("test error"))

	_, _, err := w.Hijack()

	if !errors.Is(err, ErrHijackAfterError) {
		t.Errorf("Hijack should return ErrHijackAfterError, got: %v", err)
	}
	if mock.hijacked {
		t.Error("Hijack should not delegate when error is captured")
	}
}

func TestErrorCapturingWriter_HijackReturnsErrorWhenNotSupported(t *testing.T) {
	mock := newBasicMockWriter()
	w := NewErrorCapturingWriter(mock)

	_, _, err := w.Hijack()

	if !errors.Is(err, ErrHijackNotSupported) {
		t.Errorf("Hijack should return ErrHijackNotSupported, got: %v", err)
	}
}

func TestErrorCapturingWriter_PushDelegatesToUnderlying(t *testing.T) {
	mock := newFullMockWriter()
	w := NewErrorCapturingWriter(mock)

	err := w.Push("/test", nil)

	if err != nil {
		t.Errorf("Push should succeed, got error: %v", err)
	}
	if !mock.pushed {
		t.Error("Push should delegate to underlying writer")
	}
	if mock.pushTarget != "/test" {
		t.Errorf("Push target should be /test, got %s", mock.pushTarget)
	}
}

func TestErrorCapturingWriter_PushReturnsErrorWhenErrorCaptured(t *testing.T) {
	mock := newFullMockWriter()
	w := NewErrorCapturingWriter(mock)
	w.SetError(errors.New("test error"))

	err := w.Push("/test", nil)

	if !errors.Is(err, ErrPushNotSupported) {
		t.Errorf("Push should return ErrPushNotSupported when error captured, got: %v", err)
	}
	if mock.pushed {
		t.Error("Push should not delegate when error is captured")
	}
}

func TestErrorCapturingWriter_PushReturnsErrorWhenNotSupported(t *testing.T) {
	mock := newBasicMockWriter()
	w := NewErrorCapturingWriter(mock)

	err := w.Push("/test", nil)

	if !errors.Is(err, ErrPushNotSupported) {
		t.Errorf("Push should return ErrPushNotSupported, got: %v", err)
	}
}

func TestErrorCapturingWriter_WriteDiscardsWhenErrorCaptured(t *testing.T) {
	mock := newFullMockWriter()
	w := NewErrorCapturingWriter(mock)
	w.SetError(errors.New("test error"))

	n, err := w.Write([]byte("test"))

	if err != nil {
		t.Errorf("Write should not return error, got: %v", err)
	}
	if n != 4 {
		t.Errorf("Write should return byte count, got %d", n)
	}
	if mock.Body.Len() > 0 {
		t.Error("Write should discard body when error is captured")
	}
}

func TestErrorCapturingWriter_TypeAssertions(t *testing.T) {
	mock := newFullMockWriter()
	w := NewErrorCapturingWriter(mock)

	if _, ok := interface{}(w).(http.Flusher); !ok {
		t.Error("ErrorCapturingWriter should implement http.Flusher")
	}
	if _, ok := interface{}(w).(http.Hijacker); !ok {
		t.Error("ErrorCapturingWriter should implement http.Hijacker")
	}
	if _, ok := interface{}(w).(http.Pusher); !ok {
		t.Error("ErrorCapturingWriter should implement http.Pusher")
	}
}

// --- statusRecorder Tests ---

func TestStatusRecorder_FlushDelegatesToUnderlying(t *testing.T) {
	mock := newFullMockWriter()
	r := &statusRecorder{ResponseWriter: mock}

	r.Flush()

	if !mock.flushed {
		t.Error("Flush should delegate to underlying writer")
	}
}

func TestStatusRecorder_FlushSetsWrittenAndStatusCode(t *testing.T) {
	mock := newFullMockWriter()
	r := &statusRecorder{ResponseWriter: mock}

	r.Flush()

	if !r.written {
		t.Error("Flush should set written = true")
	}
	if r.statusCode != http.StatusOK {
		t.Errorf("Flush should set statusCode to 200, got %d", r.statusCode)
	}
}

func TestStatusRecorder_HijackDelegatesToUnderlying(t *testing.T) {
	mock := newFullMockWriter()
	r := &statusRecorder{ResponseWriter: mock}

	_, _, err := r.Hijack()

	if err != nil {
		t.Errorf("Hijack should succeed, got error: %v", err)
	}
	if !mock.hijacked {
		t.Error("Hijack should delegate to underlying writer")
	}
}

func TestStatusRecorder_HijackReturnsErrorWhenNotSupported(t *testing.T) {
	mock := newBasicMockWriter()
	r := &statusRecorder{ResponseWriter: mock}

	_, _, err := r.Hijack()

	if !errors.Is(err, ErrHijackNotSupported) {
		t.Errorf("Hijack should return ErrHijackNotSupported, got: %v", err)
	}
}

func TestStatusRecorder_PushDelegatesToUnderlying(t *testing.T) {
	mock := newFullMockWriter()
	r := &statusRecorder{ResponseWriter: mock}

	err := r.Push("/test", nil)

	if err != nil {
		t.Errorf("Push should succeed, got error: %v", err)
	}
	if !mock.pushed {
		t.Error("Push should delegate to underlying writer")
	}
}

func TestStatusRecorder_PushReturnsErrorWhenNotSupported(t *testing.T) {
	mock := newBasicMockWriter()
	r := &statusRecorder{ResponseWriter: mock}

	err := r.Push("/test", nil)

	if !errors.Is(err, ErrPushNotSupported) {
		t.Errorf("Push should return ErrPushNotSupported, got: %v", err)
	}
}

func TestStatusRecorder_TypeAssertions(t *testing.T) {
	mock := newFullMockWriter()
	r := &statusRecorder{ResponseWriter: mock}

	if _, ok := interface{}(r).(http.Flusher); !ok {
		t.Error("statusRecorder should implement http.Flusher")
	}
	if _, ok := interface{}(r).(http.Hijacker); !ok {
		t.Error("statusRecorder should implement http.Hijacker")
	}
	if _, ok := interface{}(r).(http.Pusher); !ok {
		t.Error("statusRecorder should implement http.Pusher")
	}
}
