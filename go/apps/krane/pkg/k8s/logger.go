package k8s

import (
	"github.com/go-logr/logr"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

type sink struct {
	logger logging.Logger
	level  int
}

var _ logr.LogSink = (*sink)(nil)

func (s *sink) Init(info logr.RuntimeInfo) {

	//	s.logger = s.logger.WithCallDepth(info.CallDepth)

}

func (s *sink) Enabled(level int) bool {
	return level < s.level
}

func (s *sink) Info(level int, msg string, keysAndValues ...any) {
	if level < s.level {

		args := []any{}
		args = append(args, "level", level)
		args = append(args, keysAndValues...)
		s.logger.Info(msg, args...)
	}
}

func (s *sink) Error(err error, msg string, keysAndValues ...any) {
	args := []any{}
	args = append(args, "error", err.Error())
	args = append(args, keysAndValues...)

	s.logger.Error(msg, args...)
}

func (s *sink) WithValues(keysAndValues ...any) logr.LogSink {
	return &sink{logger: s.logger.With(keysAndValues...), level: s.level}
}

func (s *sink) WithName(name string) logr.LogSink {
	return &sink{logger: s.logger.With("name", name), level: s.level}
}

func CompatibleLogger(logger logging.Logger) logr.Logger {

	return logr.New(&sink{logger.WithCallDepth(5), 2})

}
