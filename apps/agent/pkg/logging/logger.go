package logging

import (
	"io"
	"os"

	"github.com/rs/zerolog"
)

type Logger = zerolog.Logger

const timeFormat = "2006-01-02T15:04:05.000MST"

type Config struct {
	Debug  bool
	Writer []io.Writer
}

func New(config *Config) Logger {
	if config == nil {
		config = &Config{}
	}
	consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: timeFormat}

	writers := []io.Writer{consoleWriter}
	if len(config.Writer) > 0 {
		writers = append(writers, config.Writer...)
	}

	multi := zerolog.MultiLevelWriter(writers...)

	logger := zerolog.New(multi).With().Timestamp().Caller().Logger()
	if config.Debug {
		logger = logger.Level(zerolog.DebugLevel)
	} else {
		logger = logger.Level(zerolog.InfoLevel)
	}
	return logger
}

func NewNoopLogger() Logger {
	return zerolog.Nop()
}
