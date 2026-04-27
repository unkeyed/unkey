package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServeWithAbortRecovery_SwallowsErrAbortHandler(t *testing.T) {
	t.Parallel()

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(http.ErrAbortHandler)
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	// Must not bubble up. If it does, the test process crashes.
	serveWithAbortRecovery(h, w, r)
}

func TestServeWithAbortRecovery_RepanicsOtherValues(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("boom")
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(sentinel)
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	defer func() {
		rec := recover()
		if rec == nil {
			t.Fatal("expected panic to propagate, got none")
		}
		if rec != sentinel {
			t.Fatalf("expected original panic value, got %v", rec)
		}
	}()

	serveWithAbortRecovery(h, w, r)
}

func TestServeWithAbortRecovery_NoPanicPassthrough(t *testing.T) {
	t.Parallel()

	called := false
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusTeapot)
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	serveWithAbortRecovery(h, w, r)

	if !called {
		t.Fatal("handler was not invoked")
	}
	if w.Code != http.StatusTeapot {
		t.Fatalf("expected status %d, got %d", http.StatusTeapot, w.Code)
	}
}
