package logging

import (
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
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

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: timeFormat})
}

func New(debug ...bool) Logger {
	isDebug := len(debug) > 0 && debug[0]
	if isDebug {
		return log.Level(zerolog.DebugLevel).With().Timestamp().Caller().Logger()
	} else {
		return log.Level(zerolog.InfoLevel).With().Timestamp().Caller().Logger()
	}
}

func NewNoopLogger() Logger {
	return zerolog.Nop()
}
