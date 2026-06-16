package logger

import (
	"context"
	"io"
	"log/slog"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
)

// ANSI SGR sequences for the pretty handler. Kept package-local so logger
// stays dependency-free; the handler is only installed when stderr is an
// interactive terminal (see newDefaultHandler), so no NO_COLOR stripping is
// needed at render time.
const (
	ansiReset  = "\033[0m"
	ansiBold   = "\033[1m"
	ansiDim    = "\033[2m"
	ansiRed    = "\033[31m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
)

// prettyHandler is a human-oriented slog.Handler for local development:
// short dim timestamps, colored levels, the message up front, and dim
// key=value attrs trailing on the same line. Machine-oriented logfmt output
// (stdlib TextHandler) remains the default for non-TTY processes.
type prettyHandler struct {
	out    io.Writer
	mu     *sync.Mutex // shared across clones so lines never interleave
	level  slog.Leveler
	prefix string   // attrs accumulated via WithAttrs, pre-rendered
	groups []string // open groups qualifying future attr keys
}

func newPrettyHandler(out io.Writer, level slog.Leveler) *prettyHandler {
	return &prettyHandler{
		out:    out,
		mu:     &sync.Mutex{},
		level:  level,
		prefix: "",
		groups: nil,
	}
}

func (h *prettyHandler) Enabled(_ context.Context, l slog.Level) bool {
	return l >= h.level.Level()
}

func (h *prettyHandler) Handle(_ context.Context, r slog.Record) error {
	b := &strings.Builder{}
	if !r.Time.IsZero() {
		b.WriteString(ansiDim + r.Time.Format("15:04:05.000") + ansiReset + " ")
	}
	b.WriteString(levelTag(r.Level))
	b.WriteString(" " + r.Message)
	b.WriteString(h.prefix)

	groupPrefix := ""
	if len(h.groups) > 0 {
		groupPrefix = strings.Join(h.groups, ".") + "."
	}
	r.Attrs(func(a slog.Attr) bool {
		appendAttr(b, groupPrefix, a)
		return true
	})

	if r.PC != 0 {
		frames := runtime.CallersFrames([]uintptr{r.PC})
		frame, _ := frames.Next()
		if frame.File != "" {
			b.WriteString(" " + ansiDim + frame.File + ":" + strconv.Itoa(frame.Line) + ansiReset)
		}
	}
	b.WriteByte('\n')

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := io.WriteString(h.out, b.String())
	return err
}

func (h *prettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h2 := h.clone()
	b := &strings.Builder{}
	groupPrefix := ""
	if len(h.groups) > 0 {
		groupPrefix = strings.Join(h.groups, ".") + "."
	}
	for _, a := range attrs {
		appendAttr(b, groupPrefix, a)
	}
	h2.prefix += b.String()
	return h2
}

func (h *prettyHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	h2 := h.clone()
	h2.groups = append(h2.groups, name)
	return h2
}

func (h *prettyHandler) clone() *prettyHandler {
	return &prettyHandler{
		out:    h.out,
		mu:     h.mu,
		level:  h.level,
		prefix: h.prefix,
		groups: slices.Clone(h.groups),
	}
}

// levelTag renders the record level as a fixed-width colored token so
// messages line up vertically across entries.
func levelTag(l slog.Level) string {
	switch {
	case l >= slog.LevelError:
		return ansiRed + ansiBold + "ERROR" + ansiReset
	case l >= slog.LevelWarn:
		return ansiYellow + "WARN " + ansiReset
	case l >= slog.LevelInfo:
		return ansiGreen + "INFO " + ansiReset
	default:
		return ansiDim + "DEBUG" + ansiReset
	}
}

// appendAttr renders one attr as a dim "key=" followed by the value, with
// group keys flattened to dotted paths. Empty attrs are skipped per the
// slog.Handler contract.
func appendAttr(b *strings.Builder, groupPrefix string, a slog.Attr) {
	a.Value = a.Value.Resolve()
	if a.Equal(slog.Attr{}) { //nolint:exhaustruct // zero attr is the sentinel per slog docs
		return
	}
	if a.Value.Kind() == slog.KindGroup {
		attrs := a.Value.Group()
		if len(attrs) == 0 {
			return
		}
		if a.Key != "" {
			groupPrefix += a.Key + "."
		}
		for _, ga := range attrs {
			appendAttr(b, groupPrefix, ga)
		}
		return
	}
	b.WriteString(" " + ansiDim + groupPrefix + a.Key + "=" + ansiReset + formatValue(a.Value))
}

// formatValue quotes values that would be ambiguous in key=value output
// (whitespace, quotes, equals signs) and passes everything else through.
func formatValue(v slog.Value) string {
	s := v.String()
	if strings.ContainsAny(s, " \t\n\"=") {
		return strconv.Quote(s)
	}
	if s == "" {
		return `""`
	}
	return s
}
