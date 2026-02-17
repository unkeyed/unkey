package config

import (
	"errors"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
)

// Validator is an optional interface for cross-field or business-rule
// validation. It is called after defaults have been applied and struct tag
// validation has passed, so implementations can assume fields are individually
// valid and defaulted.
type Validator interface {
	Validate() error
}

// Load reads a TOML configuration file at path and returns the validated
// result. Returns the zero value of T on file-read errors; delegates all
// other behavior (env expansion, defaults, validation) to [LoadBytes].
func Load[T any](path string) (T, error) {
	var zero T

	data, err := os.ReadFile(path)
	if err != nil {
		return zero, fault.Wrap(err, fault.Internal("failed to read config file: "+path))
	}

	logger.Info("using config", "path", path, "raw", string(data))

	return LoadBytes[T](data)
}

// LoadBytes parses raw TOML bytes into T, applies defaults, and validates the
// result. Useful for testing or when configuration comes from a source other
// than a file.
//
// On unmarshal or default-application failure the returned T may be partially
// populated. On validation failure the fully populated struct is returned
// alongside the error so callers can inspect partial results.
func LoadBytes[T any](data []byte) (T, error) {
	var cfg T

	expanded := os.ExpandEnv(string(data))

	if err := toml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return cfg, fault.Wrap(err, fault.Internal("failed to unmarshal TOML config"))
	}
	if err := applyDefaults(&cfg); err != nil {
		return cfg, fault.Wrap(err, fault.Internal("failed to apply defaults"))
	}

	var errs []error

	if err := validate(&cfg); err != nil {
		errs = append(errs, err)
	}

	if err := validateCustom(&cfg); err != nil {
		errs = append(errs, err)
	}

	return cfg, errors.Join(errs...)
}
