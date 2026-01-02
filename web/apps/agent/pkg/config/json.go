package config

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"os"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
	"github.com/danielgtaylor/huma/schema"
	"github.com/xeipuuv/gojsonschema"
)

func LoadFile[C any](config *C, path string) (err error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("Failed to read configuration file: %s", err)
	}

	expanded := os.ExpandEnv(string(content))

	schema, err := GenerateJsonSchema(config)
	if err != nil {
		return fmt.Errorf("Failed to generate json schema: %s", err)
	}

	v, err := gojsonschema.Validate(
		gojsonschema.NewStringLoader(schema),
		gojsonschema.NewStringLoader(expanded))
	if err != nil {
		return fmt.Errorf("Failed to validate configuration: %s", err)
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
		return fault.New(strings.Join(lines, "\n"))
	}

	err = json.Unmarshal([]byte(expanded), config)
	if err != nil {
		return fault.Wrap(err, fmsg.WithDesc("bad_config", "Failed to unmarshal configuration"))

	}
	return nil

}

// GenerateJsonSchema generates a JSON schema for the given configuration struct.
// If `file` is provided, it will be written to that file.
func GenerateJsonSchema(cfg any, file ...string) (string, error) {
	s, err := schema.Generate(reflect.TypeOf(cfg))
	if err != nil {
		return "", err
	}
	s.AdditionalProperties = true
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return "", err
	}

	if len(file) > 0 {
		err = os.WriteFile(file[0], b, 0644)
		if err != nil {
			return "", err
		}
	}

	return string(b), nil

}
