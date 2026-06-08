package logger_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/logger/loggertest"
)

func TestWideEvent_SourceIsStartCaller(t *testing.T) {
	// The wide event's source attribute should point at StartWideEvent's
	// caller (the middleware/handler that opened the event), not at the
	// End() call inside this package. Without the explicit PC capture in
	// StartWideEvent, the emitted record would carry the PC of the
	// internal Handle call in event.go and be useless for debugging.
	h := loggertest.Install(t)

	_, ev := logger.StartWideEvent(context.Background(), "incoming request")
	ev.End()

	rec := h.Last(t)
	require.NotZero(t, rec.PC, "wide event record must carry the captured PC")

	frame := loggertest.PCFrame(rec.PC)
	require.True(t, strings.HasSuffix(frame.File, "wide_test.go"),
		"source should be the test file, got %s", frame.File)
	require.Contains(t, frame.Function, "TestWideEvent_SourceIsStartCaller",
		"source function should be the test, got %s", frame.Function)
}

func TestWideEvent_NoErrors_LogsInfo(t *testing.T) {
	h := loggertest.Install(t)

	_, ev := logger.StartWideEvent(context.Background(), "request done")
	ev.End()

	rec := h.Last(t)
	require.Equal(t, "request done", rec.Message)
}

func TestWideEvent_WithError_LogsErrorWithSteps(t *testing.T) {
	h := loggertest.Install(t)

	_, ev := logger.StartWideEvent(context.Background(), "request done")
	ev.SetError(fault.New("boom"))
	ev.End()

	rec := h.Last(t)
	require.Equal(t, "error", rec.Message,
		"events with errors should be emitted under the canonical 'error' message")

	attrs := loggertest.FlatAttrs(rec)
	// The wide-event End() builds its own errors group (see event.go); the
	// faultHandler doesn't run on these because the error isn't passed as
	// a direct attr value. Make sure the existing group is preserved.
	steps, ok := attrs["errors.0.steps"].([]fault.Step)
	require.True(t, ok, "expected fault steps in errors.0.steps, got %T", attrs["errors.0.steps"])
	require.NotEmpty(t, steps)
}

func TestWideEvent_DoubleEndOnlyEmitsOnce(t *testing.T) {
	h := loggertest.Install(t)

	before := h.Snapshot()

	_, ev := logger.StartWideEvent(context.Background(), "once")
	ev.End()
	ev.End()
	ev.End()

	require.Equal(t, 1, len(h.Since(before)),
		"calling End multiple times must only emit a single record")
}
