package env

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Env struct {
	ErrorHandler func(error)
}

type EnvError struct {
	Key string
}

func (e *EnvError) Error() string {
	return fmt.Sprintf("%s is not set and no fallback provided", e.Key)
}

func (e *Env) String(key string, fallback ...string) string {
	val, err := get(key, parseString, fallback...)
	if err != nil {
		e.ErrorHandler(err)
		return val
	}
	return val
}

// Strings parses a comma-separated list of strings.
func (e *Env) Strings(key string, fallback ...[]string) []string {
	val, err := get(key, parseStrings, fallback...)
	if err != nil {
		e.ErrorHandler(err)
		return val
	}
	return val
}

// StringsAppend parses a comma-separated list of strings and appends it to the default values
func (e *Env) StringsAppend(key string, defaultValues ...[]string) []string {
	all := []string{}
	if len(defaultValues) > 0 {
		all = defaultValues[0]
	}

	if val, ok := os.LookupEnv(key); ok {
		all = append(all, strings.Split(val, ",")...)
	}

	if len(all) == 0 {
		e.ErrorHandler(&EnvError{Key: key})
		return all
	}
	return all

}

func (e *Env) Int(key string, fallback ...int) int {
	val, err := get(key, strconv.Atoi, fallback...)
	if err != nil {
		e.ErrorHandler(err)
		return val
	}
	return val
}

func (e *Env) Bool(key string, fallback ...bool) bool {
	val, err := get(key, strconv.ParseBool, fallback...)
	if err != nil {
		e.ErrorHandler(err)
		return val
	}
	return val
}

func (e *Env) Duration(key string, fallback ...time.Duration) time.Duration {
	val, err := get(key, time.ParseDuration, fallback...)
	if err != nil {
		e.ErrorHandler(err)
		return val
	}
	return val
}

func get[V any](key string, parse func(string) (V, error), fallback ...V) (V, error) {
	val, ok := os.LookupEnv(key)
	if !ok {
		if len(fallback) > 0 {
			return fallback[0], nil
		}
		return *new(V), &EnvError{Key: key}
	}

	v, err := parse(val)
	if err != nil {
		return *new(V), err
	}
	return v, nil
}

func parseString(val string) (string, error) {
	return val, nil
}

func parseStrings(val string) ([]string, error) {
	return strings.Split(val, ","), nil
}
