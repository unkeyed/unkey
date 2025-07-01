package logging

import "log"

// Logger provides a simple logging interface
type Logger interface {
	Debug(msg string, fields ...any)
	Info(msg string, fields ...any)
	Warn(msg string, fields ...any)
	Error(msg string, fields ...any)
}

// noopLogger is a logger that does nothing
type noopLogger struct{}

func (n noopLogger) Debug(msg string, fields ...any) {}
func (n noopLogger) Info(msg string, fields ...any)  {}
func (n noopLogger) Warn(msg string, fields ...any)  {}
func (n noopLogger) Error(msg string, fields ...any) {}

// stdLogger uses the standard library logger
type stdLogger struct{}

func (s stdLogger) Debug(msg string, fields ...any) {
	log.Printf("[DEBUG] %s %v", msg, fields)
}

func (s stdLogger) Info(msg string, fields ...any) {
	log.Printf("[INFO] %s %v", msg, fields)
}

func (s stdLogger) Warn(msg string, fields ...any) {
	log.Printf("[WARN] %s %v", msg, fields)
}

func (s stdLogger) Error(msg string, fields ...any) {
	log.Printf("[ERROR] %s %v", msg, fields)
}

// New creates a new logger. If opts is nil, returns a noop logger
func New(opts any) Logger {
	if opts == nil {
		return noopLogger{}
	}
	return stdLogger{}
}
