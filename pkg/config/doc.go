// Package config loads and validates TOML configuration into Go structs using
// struct tags for defaults and constraints.
//
// File-based config was chosen over CLI flags because the number of service
// options makes flag-based configuration unwieldy. TOML files give operators
// editor support, inline environment variable expansion, and validation that
// reports every error at once instead of failing on the first.
//
// # Processing Pipeline
//
// [Load] and [LoadBytes] follow the same pipeline once raw bytes are available:
// expand environment variables with [os.ExpandEnv] (supports $VAR and ${VAR}),
// unmarshal TOML into the target struct, apply default values from struct tags,
// validate struct tag constraints, and finally call [Validator].Validate if the
// type implements it.
//
// Environment variable expansion happens on the raw TOML bytes before
// unmarshalling, so references like ${DB_URL} resolve before the TOML parser
// sees them. Undefined variables expand to the empty string.
//
// Validation collects all constraint violations into a single joined error
// rather than short-circuiting on the first failure. This lets operators fix
// every problem in one pass.
//
// # Struct Tags
//
// Fields are annotated with `config:"..."` directives that control defaults
// and validation. Available directives:
//
//   - required — field must be non-zero (non-nil for pointers/slices/maps)
//   - default=V — applied when the field is the zero value after unmarshalling
//   - min=N — for numbers: minimum value; for strings/slices/maps: minimum length
//   - max=N — for numbers: maximum value; for strings/slices/maps: maximum length
//   - nonempty — strings must have length > 0; slices/maps must be non-nil and non-empty
//   - oneof=a|b|c — string must be one of the listed values
//
// Defaults are not applied to slices. Supported default types are string, int
// variants, uint variants, float variants, bool, and [time.Duration].
//
// # Usage
//
//	type Config struct {
//	    Region   string   `toml:"region"   config:"required,oneof=aws|gcp|hetzner"`
//	    HttpPort int      `toml:"httpPort" config:"default=7070,min=1,max=65535"`
//	    DbURL    string   `toml:"dbUrl"    config:"required,nonempty"`
//	    Brokers  []string `toml:"brokers"  config:"required,nonempty"`
//	}
//
//	cfg, err := config.Load[Config]("/etc/unkey/api.toml")
//
// For programmatic use or testing, [LoadBytes] accepts raw TOML bytes directly:
//
//	cfg, err := config.LoadBytes[Config](data)
//
// Types that need cross-field or semantic validation can implement [Validator];
// its Validate method is called after struct tag validation so both sources of
// errors are collected together.
package config
