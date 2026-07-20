package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/unkeyed/unkey/pkg/fault"
)

// Validator is an optional interface for cross-field or business-rule
// validation. It is called after defaults have been applied and struct tag
// validation has passed, so implementations can assume fields are individually
// valid and defaulted.
type Validator interface {
	Validate() error
}

// Load resolves a TOML configuration and returns the validated result. The
// argument is either a filesystem path to a TOML file or the raw TOML document
// itself, so a service can be configured from a mounted file or from an inline
// value such as UNKEY_CONFIG. TOML content always contains a key assignment or
// spans multiple lines while a path never does, so that split distinguishes the
// two without a sigil. All other behavior (env expansion, defaults, validation)
// is delegated to [LoadBytes].
func Load[T any](pathOrContent string) (T, error) {
	var zero T

	if !strings.ContainsAny(pathOrContent, "=\n") {
		data, err := os.ReadFile(pathOrContent)
		if err != nil {
			return zero, fault.Wrap(err, fault.Internal("failed to read config file: "+pathOrContent))
		}
		return LoadBytes[T](data)
	}

	return LoadBytes[T]([]byte(pathOrContent))
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

// Save encodes cfg as TOML and writes it to path, creating parent directories
// with mode 0700 if they don't exist. The file is written with mode 0600.
func Save[T any](path string, cfg T) (retErr error) {
	if filepath.Ext(path) != ".toml" {
		return fault.New("failed to save config: only .toml files are supported")
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fault.Wrap(err, fault.Internal("failed to create config directory"))
	}

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fault.Wrap(err, fault.Internal("failed to open config file: "+path))
	}
	defer func() {
		if closeErr := f.Close(); retErr == nil {
			retErr = closeErr
		}
	}()

	if err := toml.NewEncoder(f).Encode(cfg); err != nil {
		return fault.Wrap(err, fault.Internal("failed to encode TOML config"))
	}

	return nil
}
