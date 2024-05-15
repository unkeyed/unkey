package env

import (
	"os"
	"strconv"
)

// String returns the value of the environment variable.
//
// # If the environment variable is empty and no defaultValue is provided, it panics
//
// Example:
//
//	port := env.String("PORT", "8080")
func String(name string, defaultValue ...string) string {
	value := os.Getenv(name)
	if value != "" {
		return value
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	panic("Environment variable " + name + " is empty")
}

// Int64 returns the value of the environment variable named by the key
//
// # If the environment variable is empty and no defaultValue is provided, it panics
//
// Example:
//
//	port := env.String("PORT", "8080")
func Int64(name string, defaultValue ...int64) int64 {
	value := os.Getenv(name)
	if value != "" {
		i, ok := strconv.ParseInt(value, 10, 64)
		if ok != nil {
			panic("Environment variable " + name + " is not a valid int64")
		}
		return i
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	panic("Environment variable " + name + " is empty")
}
