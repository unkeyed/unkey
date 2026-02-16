package config

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/unkeyed/unkey/pkg/fault"
)

// envWithDefaultRe matches ${VAR:-default} patterns.
var envWithDefaultRe = regexp.MustCompile(`\$\{([^}:]+):-([^}]*)\}`)

// expandEnv expands environment variable references in s. It supports $VAR,
// ${VAR} (via os.Expand), and ${VAR:-default} where default is used when VAR
// is unset or empty.
func expandEnv(s string) string {
	// First pass: replace ${VAR:-default} with the env value or the default.
	s = envWithDefaultRe.ReplaceAllStringFunc(s, func(match string) string {
		parts := envWithDefaultRe.FindStringSubmatch(match)
		if val, ok := os.LookupEnv(parts[1]); ok && val != "" {
			return val
		}
		return parts[2]
	})
	// Second pass: expand remaining $VAR and ${VAR} references.
	return os.ExpandEnv(s)
}

// Format represents the encoding format of a configuration file. When using
// Load, the format is auto-detected from the file extension; when using
// LoadBytes, the caller specifies the format explicitly.
type Format int

const (
	// TOML indicates TOML encoding.
	TOML Format = iota
)

// Validator is an optional interface that configuration types can implement
// to provide custom validation logic beyond struct tag validation. Validate
// is called after defaults have been applied and struct tag validation has
// passed, so implementations can assume the struct is fully populated with
// at least its default values and individually valid fields. Validate should
// return an error describing any cross-field or business-rule violations.
type Validator interface {
	Validate() error
}

// Load reads a configuration file at path, parses it into T, and returns the
// validated result. The format is detected from the file extension: ".toml" for
// TOML. An unsupported extension returns an error.
//
// The processing pipeline is: read the file, expand environment variables in
// the raw bytes (supporting $VAR, ${VAR}, and ${VAR:-default} syntax via
// expandEnv), unmarshal into T, apply struct tag defaults, run struct tag
// validation, and finally call T.Validate if T implements Validator.
//
// Load returns an error if the file cannot be read, the extension is
// unsupported, or unmarshaling fails; in all of these cases the zero value of
// T is returned. If validation fails, Load returns the fully populated struct
// alongside the error so callers can inspect partial results. Validation
// collects all tag and Validator errors rather than failing fast, joining them
// via errors.Join.
func Load[T any](path string) (T, error) {
	var zero T

	ext := strings.ToLower(filepath.Ext(path))

	var format Format
	switch ext {
	case ".toml":
		format = TOML
	default:
		return zero, fault.New("unsupported config file extension",
			fault.Internal("extension: "+ext),
		)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return zero, fault.Wrap(err, fault.Internal("failed to read config file: "+path))
	}

	return LoadBytes[T](data, format)
}

// LoadBytes parses raw configuration bytes in the given format, applies
// defaults, and validates the result. This is useful for testing or when
// configuration comes from sources other than files (environment variables,
// embedded resources, remote stores).
//
// The processing pipeline is: expand environment variables in data (supporting
// $VAR, ${VAR}, and ${VAR:-default} syntax via expandEnv), unmarshal into T,
// apply struct tag defaults, run struct tag validation, and finally call
// T.Validate if T implements Validator.
//
// On unmarshal failure, LoadBytes returns the zero value of T and an error. On
// validation failure, LoadBytes returns the populated struct alongside the
// error so callers can inspect partial results. Validation collects all tag and
// Validator errors rather than failing fast, joining them via errors.Join.
func LoadBytes[T any](data []byte, format Format) (T, error) {
	var cfg T

	expanded := expandEnv(string(data))

	switch format {
	case TOML:
		if err := toml.Unmarshal([]byte(expanded), &cfg); err != nil {
			return cfg, fault.Wrap(err, fault.Internal("failed to unmarshal TOML config"))
		}
	default:
		return cfg, fault.New("unknown config format")
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
