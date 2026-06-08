// Package loggertest provides shared testing utilities for the global
// [logger] package: a slog handler that records every emitted record and
// helpers for inspecting them.
//
// This is testing-only — it lives in its own package so production code
// can't accidentally depend on it, but assertions across services can
// reuse the same capture implementation instead of redefining one in
// every test file.
package loggertest

import (
	"context"
	"log/slog"
	"runtime"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/logger"
)

// PCFrame turns a single program counter into its first stack frame.
// Useful for asserting on the source attribute slog will derive from a
// record's PC.
func PCFrame(pc uintptr) runtime.Frame {
	frames := runtime.CallersFrames([]uintptr{pc})
	f, _ := frames.Next()
	return f
}

// CaptureHandler is a [slog.Handler] that stores every log record it
// receives. Install it into the global logger via [Install] (or call
// [logger.AddHandler] directly) and inspect the captured records in
// assertions.
//
// Safe for concurrent use.
type CaptureHandler struct {
	mu      sync.Mutex
	records []slog.Record
}

// New returns an empty CaptureHandler. Most tests should use [Install]
// instead, which both constructs the handler and wires it into the
// global logger.
func New() *CaptureHandler {
	return &CaptureHandler{mu: sync.Mutex{}, records: nil}
}

// Install registers a fresh CaptureHandler with the global logger and
// returns it. Every subsequent log call (via logger.Error, slog.Info,
// wide events, etc.) will be recorded.
//
// The global logger keeps a reference to the handler for the rest of
// the process; tests should not rely on cleanup between runs. Use
// [CaptureHandler.Records] and [CaptureHandler.Snapshot] to scope
// assertions to records emitted after a known point.
func Install(t *testing.T) *CaptureHandler {
	t.Helper()
	h := New()
	logger.AddHandler(h)
	return h
}

func (h *CaptureHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }

func (h *CaptureHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	// Clone so later mutations on the record (e.g. AddAttrs by downstream
	// handlers in the same fan-out) don't bleed into our snapshot.
	h.records = append(h.records, r.Clone())
	return nil
}

func (h *CaptureHandler) WithAttrs(_ []slog.Attr) slog.Handler { return h }
func (h *CaptureHandler) WithGroup(_ string) slog.Handler      { return h }

// Records returns a copy of every record captured so far.
func (h *CaptureHandler) Records() []slog.Record {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]slog.Record, len(h.records))
	copy(out, h.records)
	return out
}

// Snapshot returns the current record count. Pair with [CaptureHandler.Since]
// to assert only on records emitted by a specific block of test code, which
// matters because the global logger is shared across tests.
func (h *CaptureHandler) Snapshot() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.records)
}

// Since returns all records captured after the given snapshot index.
func (h *CaptureHandler) Since(idx int) []slog.Record {
	h.mu.Lock()
	defer h.mu.Unlock()
	if idx > len(h.records) {
		return nil
	}
	out := make([]slog.Record, len(h.records)-idx)
	copy(out, h.records[idx:])
	return out
}

// Last returns the most recent record, failing the test if none have
// been captured.
func (h *CaptureHandler) Last(t *testing.T) slog.Record {
	t.Helper()
	h.mu.Lock()
	defer h.mu.Unlock()
	require.NotEmpty(t, h.records, "expected at least one log record")
	return h.records[len(h.records)-1]
}

// Find returns the first captured record whose message equals msg. Tests
// use this instead of indexing when the global logger is shared and the
// order of records isn't predictable.
func (h *CaptureHandler) Find(t *testing.T, msg string) slog.Record {
	t.Helper()
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, r := range h.records {
		if r.Message == msg {
			return r
		}
	}
	t.Fatalf("no record with message %q (captured %d records)", msg, len(h.records))
	return slog.Record{} //nolint:exhaustruct // unreachable; t.Fatalf aborts the test
}

// FlatAttrs collapses a record's attributes into key→value, flattening
// groups with dotted keys (e.g. "http.method"). Keeps test assertions
// readable without having to manually walk slog.Value.Group() trees.
func FlatAttrs(r slog.Record) map[string]any {
	out := map[string]any{}
	var walk func(prefix string, attrs []slog.Attr)
	walk = func(prefix string, attrs []slog.Attr) {
		for _, a := range attrs {
			key := prefix + a.Key
			if a.Value.Kind() == slog.KindGroup {
				walk(key+".", a.Value.Group())
				continue
			}
			out[key] = a.Value.Any()
		}
	}
	r.Attrs(func(a slog.Attr) bool {
		if a.Value.Kind() == slog.KindGroup {
			walk(a.Key+".", a.Value.Group())
		} else {
			out[a.Key] = a.Value.Any()
		}
		return true
	})
	return out
}
