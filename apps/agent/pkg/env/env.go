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

func (e *Env) String(name string, fallback ...string) string {
	value := os.Getenv(name)
	if value != "" {
		return value
	}
	if len(fallback) > 0 {
		return fallback[0]
	}
	e.ErrorHandler(fmt.Errorf("%s is not set and no fallback provided", name))
	return ""
}

// Strings parses a comma-separated list of strings.
func (e *Env) Strings(name string, fallback ...[]string) []string {
	value := os.Getenv(name)
	if value != "" {
		return strings.Split(value, ",")
	}
	if len(fallback) > 0 {
		return fallback[0]
	}
	e.ErrorHandler(fmt.Errorf("%s is not set and no fallback provided", name))
	return []string{}

}

// Strings parses a comma-separated list of strings and appends it to the default values
func (e *Env) StringsAppend(name string, defaultValues ...[]string) []string {
	all := []string{}
	if len(defaultValues) > 0 {
		all = defaultValues[0]
	}

	value := os.Getenv(name)
	if value != "" {
		all = append(all, strings.Split(value, ",")...)
	}
	if len(all) == 0 {
		e.ErrorHandler(fmt.Errorf("%s is not set and no fallback provided", name))
		return []string{}
	}
	return all

}

func (e *Env) Int(name string, fallback ...int) int {
	value := os.Getenv(name)
	if value != "" {
		i, err := strconv.Atoi(value)
		if err != nil {
			e.ErrorHandler(err)
			return 0
		}
		return i
	}
	if len(fallback) > 0 {
		return fallback[0]
	}
	e.ErrorHandler(fmt.Errorf("%s is not set and no fallback provided", name))
	return 0
}

func (e *Env) Bool(name string, fallback ...bool) bool {
	value := os.Getenv(name)
	if value != "" {
		b, err := strconv.ParseBool(value)
		if err != nil {
			e.ErrorHandler(err)
			return false
		}
		return b
	}
	if len(fallback) > 0 {
		return fallback[0]
	}
	e.ErrorHandler(fmt.Errorf("%s is not set and no fallback provided", name))
	return false
}

func (e *Env) Duration(name string, fallback ...time.Duration) time.Duration {
	value := os.Getenv(name)
	if value != "" {
		d, err := time.ParseDuration(value)
		if err != nil {
			e.ErrorHandler(err)
			return 0
		}
		return d
	}
	if len(fallback) > 0 {
		return fallback[0]
	}
	e.ErrorHandler(fmt.Errorf("%s is not set and no fallback provided", name))
	return 0
}
