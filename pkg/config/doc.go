// Package config loads, validates, and defaults struct-tag-driven configuration
// from TOML files with environment variable expansion.
//
// Configuration is currently wired through CLI flags (pkg/cli), but as the
// number of options grows, file-based config provides better ergonomics:
// editors gain autocomplete and validation via JSON Schema,
// environment variables are expanded inline, and validation reports every
// error at once instead of failing on the first one.
//
// # Processing Pipeline
//
// Both [Load] and [LoadBytes] follow the same pipeline once raw bytes are
// available: expand environment variables with os.ExpandEnv, unmarshal into
// the target struct, apply default values from struct tags, validate struct
// tag constraints, and finally call the [Validator] interface if the type
// implements it.
//
// Environment variable expansion happens on the raw bytes before unmarshalling
// so that references like ${DB_URL} or shell-style defaults like ${PORT:-8080}
// work directly inside TOML values without any awareness from the
// unmarshaller.
//
// Validation collects all constraint violations into a single error rather
// than short-circuiting on the first failure. This lets operators (and CI)
// fix every problem in one pass.
//
// # Struct Tags
//
// Fields are annotated with `config:"..."` directives that control defaults
// and validation. Available directives: required, default=V, min=N, max=N,
// nonempty, and oneof=a|b|c. min/max check the numeric value for numbers
// and the length for strings, slices, and maps.
//
// # Formats
//
// [Load] auto-detects the format from the file extension: ".toml" for TOML.
// [LoadBytes] accepts an explicit [Format] constant.
//
// TOML uses github.com/BurntSushi/toml for full TOML v1.0.0 support.
// Nested Go structs map to TOML [section] table headers. For example:
//
//	type Config struct {
//	    Database DB `toml:"database"`
//	}
//	type DB struct {
//	    Host string `toml:"host"`
//	}
//
// maps to:
//
//	[database]
//	host = "localhost"
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
// For programmatic use or testing, [LoadBytes] accepts raw bytes with an
// explicit [Format]:
//
//	cfg, err := config.LoadBytes[Config](data, config.TOML)
//
// Types that need cross-field or semantic validation can implement [Validator];
// its Validate method is called after struct tag validation so both sources of
// errors are collected together.
package config
