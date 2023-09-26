package logging

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/rs/zerolog"
)

type Logger = zerolog.Logger

const timeFormat = "2006-01-02T15:04:05.000MST"

func init() {
	_, file, _, _ := runtime.Caller(0)

	dir := file
	for i := 0; i < 3; i++ {
		dir = filepath.Dir(dir)
	}
	prefixPath := fmt.Sprintf("%s/", filepath.ToSlash(dir))

	zerolog.CallerMarshalFunc = func(pc uintptr, file string, line int) string {
		return fmt.Sprintf("%s:%s", strings.TrimPrefix(file, prefixPath), strconv.Itoa(line))
	}

	zerolog.TimeFieldFormat = timeFormat

}

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
