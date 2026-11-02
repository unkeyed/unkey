package guidance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/prompt"
	"golang.org/x/term"
)

// Attach enables guided input only on canonical resource commands.
func Attach(command *cli.Command, legacy string) {
	exclusive := []string{"no-input"}
	for _, flag := range command.Flags {
		if flag.Name() == "body" {
			exclusive = append(exclusive, "body")
		}
	}
	command.Flags = append(command.Flags,
		cli.Bool("interactive", "Choose inputs and optional settings interactively", cli.MutuallyExclusive(exclusive...)),
		cli.Bool("no-input", "Never prompt; missing inputs fail immediately"),
	)
	command.Prepare = func(ctx context.Context, cmd *cli.Command) error {
		if cmd.Bool("no-input") || cmd.FlagIsSet("body") || (cmd.FlagIsSet("interactive") && !cmd.Bool("interactive")) {
			return nil
		}
		schema, ok := Lookup(legacy)
		if !ok {
			return fmt.Errorf("guided input is not declared for %s", legacy)
		}
		if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stderr.Fd())) {
			if cmd.Bool("interactive") {
				return fmt.Errorf("--interactive requires a terminal on stdin and stderr; use flags or --body in scripts")
			}
			return nil
		}
		array := schema.Type == "array"
		if array {
			var wrapper Schema
			wrapper.Type = "object"
			wrapper.Properties = map[string]*Schema{"limits": schema}
			wrapper.Required = []string{"limits"}
			schema = &wrapper
		}
		values, err := flagValues(cmd, schema, legacy)
		if err != nil {
			return err
		}
		update := strings.Contains(legacy, " update-")
		needsChanges := update && !hasChanges(schema, values)
		if !cmd.Bool("interactive") && !missingRequired(schema, values) && !needsChanges {
			return nil
		}
		if len(schema.Properties) == 0 && len(schema.Alternatives) == 0 {
			return nil
		}
		conflicts := map[string][]string{}
		for _, flag := range cmd.Flags {
			name, field := fieldForFlag(schema, flag.Name())
			if field == nil {
				continue
			}
			for _, other := range cmd.ConflictingFlags(flag.Name()) {
				if otherName, otherField := fieldForFlag(schema, other); otherField != nil {
					conflicts[name] = append(conflicts[name], otherName)
				}
			}
		}
		if legacy == "gateway update-policy" {
			rules := []string{"keyauth", "ratelimit", "firewall", "openapi", "logging"}
			for _, rule := range rules {
				conflicts[rule] = rules
			}
		}
		form := form{
			prompt:    prompt.New(prompt.WithWriter(os.Stderr)),
			out:       os.Stderr,
			pick:      resourcePicker(ctx, cmd, legacy),
			conflicts: conflicts,
		}
		if _, err := fmt.Fprintln(form.out, "Guided input · Ctrl+C cancels without sending the request."); err != nil {
			return err
		}
		if err := form.object(ctx, schema, values, "", needsChanges); err != nil {
			return fmt.Errorf("guided input cancelled: %w", err)
		}
		var request any = values
		if array {
			request = values["limits"]
		}
		body, err := json.Marshal(request)
		if err != nil {
			return err
		}
		bodyFlag := cli.String("body", "Guided request body")
		if err := bodyFlag.Parse(string(body)); err != nil {
			return err
		}
		flags := []cli.Flag{bodyFlag}
		for _, flag := range cmd.Flags {
			if commonFlag(flag.Name()) && flag.Name() != "interactive" {
				flags = append(flags, flag)
			}
		}
		cmd.Flags = flags
		cmd.RequireOneOf = nil
		return nil
	}
}

func commonFlag(name string) bool {
	switch name {
	case "root-key", "api-url", "config", "output", "json", "debug", "yes", "interactive", "no-input", "error-format", "reveal":
		return true
	default:
		return false
	}
}

func flagValues(cmd *cli.Command, schema *Schema, legacy string) (map[string]any, error) {
	values := map[string]any{}
	for _, flag := range cmd.Flags {
		if !flag.IsSet() || commonFlag(flag.Name()) || flag.Name() == "body" {
			continue
		}
		if legacy == "gateway update-policy" && flag.Name() == "policy" {
			value, err := decodeValue([]byte(cmd.String("policy")))
			if err != nil {
				return nil, fmt.Errorf("invalid JSON for --policy: %w", err)
			}
			fields, ok := value.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("--policy must be a JSON object")
			}
			for name, value := range fields {
				if schema.Properties[name] == nil || targetField(name) {
					return nil, fmt.Errorf("unknown update field %q in --policy", name)
				}
				values[name] = value
			}
			continue
		}
		name, field := fieldForFlag(schema, flag.Name())
		if field == nil {
			return nil, fmt.Errorf("no guided request field for --%s", flag.Name())
		}
		var value any
		switch flag := flag.(type) {
		case *cli.StringFlag:
			value = flag.Value()
			if field.Type != "string" || len(field.Alternatives) > 0 {
				var err error
				value, err = decodeValue([]byte(flag.Value()))
				if err != nil {
					return nil, fmt.Errorf("invalid JSON for --%s: %w", flag.Name(), err)
				}
			}
		case *cli.EnumFlag:
			value = flag.Value()
		case *cli.BoolFlag:
			value = flag.Value()
		case *cli.IntFlag:
			value = flag.Value()
		case *cli.Int64Flag:
			value = flag.Value()
		case *cli.FloatFlag:
			value = flag.Value()
		case *cli.StringSliceFlag:
			value = flag.Value()
		default:
			return nil, fmt.Errorf("unsupported guided flag --%s", flag.Name())
		}
		values[name] = value
	}
	return values, nil
}

func fieldForFlag(schema *Schema, flag string) (string, *Schema) {
	for name, field := range schema.Properties {
		if strings.EqualFold(name, strings.ReplaceAll(flag, "-", "")) {
			return name, field
		}
	}
	return "", nil
}

func missingRequired(schema *Schema, values map[string]any) bool {
	for _, name := range schema.Required {
		if value, ok := values[name]; !ok || value == "" {
			return true
		}
	}
	if len(schema.Alternatives) > 0 {
		for _, alternative := range schema.Alternatives {
			if !missingRequired(alternative, values) {
				return false
			}
		}
		return true
	}
	return false
}

func hasChanges(schema *Schema, values map[string]any) bool {
	for name := range values {
		if !targetField(name) {
			return true
		}
	}
	return len(schema.Properties) == 0
}

func targetField(name string) bool {
	switch name {
	case "project", "app", "environment", "keyId", "apiId", "permission", "role", "identity", "namespace", "portal", "permissionId", "roleId", "identityId", "namespaceId", "overrideId", "portalId", "deploymentId", "domainId", "policyId", "id":
		return true
	default:
		return false
	}
}

func decodeValue(raw []byte) (any, error) {
	var value any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, fmt.Errorf("expected one JSON value")
	}
	return value, nil
}
