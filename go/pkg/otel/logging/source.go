package logging

import (
	"fmt"
	"log/slog"
	"runtime"
)

func withSource(args []any) []any {
	_, file, line, ok := runtime.Caller(2)

	if !ok {
		return args
	}

	return append(args, slog.Attr{
		Key:   "source",
		Value: slog.AnyValue(fmt.Sprintf("%s:%d", file, line))})
}
