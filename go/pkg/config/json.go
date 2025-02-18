package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"os"

	"github.com/danielgtaylor/huma/schema"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/xeipuuv/gojsonschema"
)

// LoadFile reads a file and populates the provided `config`.
//
// An error is returnd if the file doesn't exist or doesn't conform to the
// schema.
func LoadFile[C any](config *C, path string) (err error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("Failed to read configuration file: %w", err)
	}

	expanded := os.ExpandEnv(string(content))

	schema, err := GenerateJsonSchema(config)
	if err != nil {
		return fmt.Errorf("Failed to generate json schema: %w", err)
	}

	v, err := gojsonschema.Validate(
		gojsonschema.NewStringLoader(schema),
		gojsonschema.NewStringLoader(expanded))
	if err != nil {
		return fmt.Errorf("Failed to validate configuration: %w", err)
	}

	if !v.Valid() {
		lines := []string{"Configuration is invalid", fmt.Sprintf("read file: %s", path), ""}

		for _, e := range v.Errors() {
			lines = append(lines, fmt.Sprintf("  - %s: %s", e.Field(), e.Description()))
		}
		lines = append(lines, "")
		lines = append(lines, "")
		lines = append(lines, "Configuration received:")
		lines = append(lines, expanded)

		message := strings.Join(lines, "\n")
		return errors.New(message)
	}

	err = json.Unmarshal([]byte(expanded), config)
	if err != nil {
		return fault.Wrap(err, fault.WithDesc("bad_config", "Failed to unmarshal configuration"))

	}
	return nil

}

// GenerateJsonSchema generates a JSON schema for the given configuration struct.
// If `file` is provided, it will be written to that file.
func GenerateJsonSchema(cfg any, file ...string) (string, error) {
	s, err := schema.Generate(reflect.TypeOf(cfg))
	if err != nil {
		return "", fault.Wrap(err, fault.WithDesc("unable to generate schema", ""))
	}
	s.AdditionalProperties = true
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return "", fault.Wrap(err, fault.WithDesc("unable to marshal schema", ""))
	}

	if len(file) > 0 {
		err = os.WriteFile(file[0], b, 0600)
		if err != nil {
			return "", fault.Wrap(err, fault.WithDesc("unable to write file", ""))
		}
	}

	return string(b), nil

}
