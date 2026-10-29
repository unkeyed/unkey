package discovery

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/unkeyed/unkey/pkg/cli"
)

func describeFlags(command *cli.Command, request Request, legacy string) ([]Flag, error) {
	body, err := request.object(request.RequestBody)
	if err != nil {
		return nil, err
	}
	var content map[string]struct {
		Schema json.RawMessage `json:"schema"`
	}
	if body["content"] != nil {
		if err := json.Unmarshal(body["content"], &content); err != nil {
			return nil, err
		}
	}
	schema := content["application/json"].Schema
	properties := map[string]json.RawMessage{}
	if len(schema) > 0 {
		root, err := request.object(schema)
		if err != nil {
			return nil, err
		}
		if root["properties"] != nil {
			if err := json.Unmarshal(root["properties"], &properties); err != nil {
				return nil, err
			}
		}
	}
	flags := make([]Flag, 0, len(command.Flags))
	for _, flag := range command.Flags {
		entry, err := flagMetadata(flag)
		if err != nil {
			return nil, err
		}
		entry.Conflicts = command.ConflictingFlags(flag.Name())
		slices.Sort(entry.Conflicts)
		if entry.Required {
			entry.RequiredUnless = entry.Conflicts
		}
		for name, property := range properties {
			if strings.EqualFold(name, strings.ReplaceAll(flag.Name(), "-", "")) {
				entry.RequestFields = []string{name}
				entry.Schema = property
				break
			}
		}
		if flag.Name() == "body" || (legacy == "ratelimit multi-limit" && flag.Name() == "limits") {
			entry.Schema = schema
			entry.Encoding = "json"
		}
		if legacy == "gateway update-policy" && flag.Name() == "policy" {
			if err := policyProjection(&entry, command, properties); err != nil {
				return nil, err
			}
		}
		flags = append(flags, entry)
	}
	return flags, nil
}

func flagMetadata(flag cli.Flag) (Flag, error) {
	var entry Flag
	entry.Name = flag.Name()
	entry.Usage = flag.Usage()
	entry.Required = flag.Required()
	if environment, ok := flag.(interface{ EnvVar() string }); ok {
		entry.Environment = environment.EnvVar()
	}
	switch flag := flag.(type) {
	case *cli.StringFlag:
		entry.Type = "string"
		if strings.Contains(flag.Usage(), "JSON") {
			entry.Encoding = "json"
		}
	case *cli.EnumFlag:
		entry.Type = "string"
		entry.Enum = flag.Allowed()
	case *cli.BoolFlag:
		entry.Type = "boolean"
	case *cli.IntFlag, *cli.Int64Flag:
		entry.Type = "integer"
	case *cli.FloatFlag:
		entry.Type = "number"
	case *cli.StringSliceFlag:
		entry.Type = "string[]"
		entry.Encoding = "comma-separated or repeated"
	case *cli.DurationFlag:
		entry.Type = "duration"
	default:
		return Flag{}, fmt.Errorf("unsupported flag type for --%s", flag.Name())
	}
	return entry, nil
}

func policyProjection(entry *Flag, command *cli.Command, properties map[string]json.RawMessage) error {
	fields := maps.Clone(properties)
	for _, flag := range command.Flags {
		for name := range fields {
			if strings.EqualFold(name, strings.ReplaceAll(flag.Name(), "-", "")) {
				delete(fields, name)
			}
		}
	}
	schema, err := json.Marshal(struct {
		Type                 string                     `json:"type"`
		Properties           map[string]json.RawMessage `json:"properties"`
		AdditionalProperties bool                       `json:"additionalProperties"`
		MinProperties        int                        `json:"minProperties"`
	}{Type: "object", Properties: fields, AdditionalProperties: false, MinProperties: 1})
	if err != nil {
		return err
	}
	entry.Encoding = "json"
	entry.RequestFields = slices.Sorted(maps.Keys(fields))
	entry.Schema = schema
	entry.AtMostOneField = []string{"keyauth", "ratelimit", "firewall", "openapi", "logging"}
	return nil
}
