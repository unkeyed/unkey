package logger_test

import (
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/logger/loggertest"
)

func TestFaultHandler_EnrichesFaultError(t *testing.T) {
	h := loggertest.Install(t)

	root := fault.New("root cause", fault.Code(codes.App.Internal.UnexpectedError.URN()))
	wrapped := fault.Wrap(root, fault.Internal("outer"))

	logger.Error("boom", "error", wrapped)

	attrs := loggertest.FlatAttrs(h.Last(t))

	steps, ok := attrs["error.steps"].([]fault.Step)
	require.True(t, ok, "expected []fault.Step for error.steps, got %T", attrs["error.steps"])
	require.Len(t, steps, 3, "fault chain should produce 3 steps (root + Wrap-merge + outer)")
	require.Equal(t, "root cause", steps[0].Message)
	require.Equal(t, "outer", steps[len(steps)-1].Message)

	loc, ok := attrs["error.location"].(string)
	require.True(t, ok, "error.location should be a string")
	require.Equal(t, steps[len(steps)-1].Location, loc,
		"error.location should match the outermost wrap")
}

func TestFaultHandler_IgnoresNonFaultError(t *testing.T) {
	h := loggertest.Install(t)

	logger.Error("boom", "error", errors.New("plain error"))

	attrs := loggertest.FlatAttrs(h.Last(t))
	_, hasSteps := attrs["error.steps"]
	_, hasLoc := attrs["error.location"]
	require.False(t, hasSteps, "stdlib errors should not produce error.steps")
	require.False(t, hasLoc, "stdlib errors should not produce error.location")
}

func TestFaultHandler_IgnoresWhenNoErrorArg(t *testing.T) {
	h := loggertest.Install(t)

	logger.Info("hello", "user_id", "u_123", "count", 42)

	attrs := loggertest.FlatAttrs(h.Last(t))
	_, hasSteps := attrs["error.steps"]
	require.False(t, hasSteps, "records without an error value should not be enriched")
	require.Equal(t, "u_123", attrs["user_id"])
}

func TestFaultHandler_FirstFaultErrorWins(t *testing.T) {
	h := loggertest.Install(t)

	first := fault.New("first error")
	second := fault.New("second error")

	logger.Error("boom", "first", first, "second", second)

	attrs := loggertest.FlatAttrs(h.Last(t))
	steps := attrs["error.steps"].([]fault.Step)
	require.Equal(t, "first error", steps[0].Message,
		"only the first fault error in args should drive enrichment")
}

func TestFaultHandler_AppliesAcrossAllSinks(t *testing.T) {
	// Two captures installed back-to-back must BOTH see the enriched
	// record — proves enrichment runs at the top of the fan-out instead
	// of being baked into a single inner handler.
	a := loggertest.Install(t)
	b := loggertest.Install(t)

	logger.Error("boom", "error", fault.New("oops"))

	for name, h := range map[string]*loggertest.CaptureHandler{"a": a, "b": b} {
		attrs := loggertest.FlatAttrs(h.Last(t))
		_, ok := attrs["error.steps"]
		require.True(t, ok, "handler %s should have received the enriched record", name)
	}
}

func TestFaultHandler_FaultInSlogAttrValue(t *testing.T) {
	h := loggertest.Install(t)

	err := fault.New("via attr")
	logger.Error("boom", slog.Any("err", err))

	attrs := loggertest.FlatAttrs(h.Last(t))
	require.NotNil(t, attrs["error.steps"],
		"errors passed via slog.Any(...) should still be detected")
}

func TestLoggerAliases_SourceIsCaller(t *testing.T) {
	// Aliasing logger.Error -> slog.Error (no wrapper frame) means the PC
	// captured by stdlib slog points at the caller, not at this package.
	h := loggertest.Install(t)

	logger.Error("boom") // <-- expected source line

	rec := h.Last(t)
	require.NotZero(t, rec.PC, "record PC must be set")

	frame := loggertest.PCFrame(rec.PC)
	require.True(t, strings.HasSuffix(frame.File, "fault_handler_test.go"),
		"source file should be the test file, got %s", frame.File)
	require.Contains(t, frame.Function, "TestLoggerAliases_SourceIsCaller",
		"source function should be the test, got %s", frame.Function)
}
