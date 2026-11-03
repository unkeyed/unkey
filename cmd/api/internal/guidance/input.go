package guidance

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
	"golang.org/x/term"
)

// Attach enables interactive prompts only on canonical resource commands.
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
	command.Prepare = func(ctx context.Context, cmd *cli.Command) (err error) {
		if cmd.Bool("no-input") || cmd.FlagIsSet("body") || (cmd.FlagIsSet("interactive") && !cmd.Bool("interactive")) {
			return nil
		}
		schema, ok := Lookup(legacy)
		if !ok {
			return fmt.Errorf("prompts are not declared for %s", legacy)
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
		_, operation, _ := strings.Cut(legacy, " ")
		read := strings.HasPrefix(operation, "get-") || strings.HasPrefix(operation, "list-") || legacy == "keys whoami"
		destructive := strings.HasPrefix(operation, "delete-") || strings.HasPrefix(operation, "remove-")
		form := form{
			prompt:        nil,
			out:           os.Stderr,
			screen:        nil,
			pick:          resourcePicker(ctx, cmd, legacy),
			fields:        nil,
			conflicts:     conflicts,
			selections:    nil,
			reviewPending: !read,
		}
		if os.Getenv("TERM") == "dumb" || !terminalInputSupported {
			form.prompt = newLineForm()
			if _, err := fmt.Fprintln(form.out, cancelHint); err != nil {
				return err
			}
		} else {
			screen := newTerminalForm()
			defer func() {
				if err != nil || !destructive {
					err = errors.Join(err, screen.finish())
				}
			}()
			form.prompt, form.out, form.screen = screen, screen, screen
			form.show(schema, values, "")
		}
		if err := form.object(ctx, schema, values, "", needsChanges); err != nil {
			return cancelled(err, "cancelled; no request was sent")
		}
		if !read && !destructive {
			if err := form.reviewMenu(ctx, schema, values, "", needsChanges, reviewMode{
				action: cmd.Usage, warning: "", expanded: needsChanges || cmd.Bool("interactive"), destructive: false,
			}); err != nil {
				return cancelled(err, "cancelled; no request was sent")
			}
		}
		body, err := requestJSON(values, array)
		if err != nil {
			return err
		}
		bodyFlag := cli.String("body", "Request body from prompts")
		if err := bodyFlag.Parse(string(body)); err != nil {
			return err
		}
		flags := []cli.Flag{bodyFlag}
		for _, flag := range cmd.Flags {
			if util.IsCommonFlag(flag.Name()) && flag.Name() != "interactive" {
				flags = append(flags, flag)
			}
		}
		cmd.Flags = flags
		cmd.RequireOneOf = nil
		action := cmd.Action
		cmd.Action = func(ctx context.Context, cmd *cli.Command) error {
			labels := make(map[string]util.ResourceLabel, len(form.selections))
			for field, selection := range form.selections {
				value := strings.TrimPrefix(selection.Key, "id:")
				if values[field] == value {
					labels[field] = util.ResourceLabel{Value: value, Label: selection.Label}
				}
			}
			return action(util.WithResourceLabels(ctx, labels), cmd)
		}
		if destructive {
			cmd.Action = form.destructiveAction(cmd.Action, bodyFlag, schema, values, array, needsChanges)
		}
		cmd.Action = form.recoverValidation(cmd.Action, bodyFlag, schema, values, array, needsChanges, destructive)
		return nil
	}
}

func (f *form) destructiveAction(action cli.Action, body *cli.StringFlag, schema *Schema, values map[string]any, array, needsChanges bool) cli.Action {
	return func(ctx context.Context, cmd *cli.Command) (err error) {
		if f.screen != nil {
			defer func() { err = errors.Join(err, f.screen.finish()) }()
		}
		ctx = util.WithConfirmation(ctx, func(warning string) error {
			for field, selection := range f.selections {
				id := strings.TrimPrefix(selection.Key, "id:")
				if values[field] == id {
					warning = strings.ReplaceAll(warning, strconv.Quote(id), strconv.Quote(selection.Label))
				}
			}
			hideWarning := func() {}
			if f.screen != nil {
				hideWarning = f.section("confirmation", func(p palette) string {
					return p.mark("!") + " " + warning + "\n"
				})
			}
			err := f.reviewMenu(ctx, schema, values, "", needsChanges, reviewMode{action: cmd.Usage, warning: warning, expanded: false, destructive: true})
			hideWarning()
			if err != nil {
				return cancelled(err, "cancelled; no request was sent")
			}
			if f.screen != nil {
				return f.screen.finish()
			}
			return nil
		})
		for {
			err := action(ctx, cmd)
			if !errors.Is(err, errReviewEdited) {
				return err
			}
			updated, err := requestJSON(values, array)
			if err != nil {
				return err
			}
			if err := body.Parse(string(updated)); err != nil {
				return err
			}
		}
	}
}

// cancelled replaces interruption causes with message so the user sees what was not sent.
func cancelled(err error, message string) error {
	if errors.Is(err, errInterrupted) || errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
		return &util.CancellationError{Message: message}
	}
	return err
}

func requestJSON(values map[string]any, array bool) ([]byte, error) {
	if array {
		return json.Marshal(values["limits"])
	}
	return json.Marshal(values)
}

func flagValues(cmd *cli.Command, schema *Schema, legacy string) (map[string]any, error) {
	values := map[string]any{}
	for _, flag := range cmd.Flags {
		if !flag.IsSet() || util.IsCommonFlag(flag.Name()) || flag.Name() == "body" {
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
			return nil, fmt.Errorf("no prompt field for --%s", flag.Name())
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
			items := make([]any, len(flag.Value()))
			for i, item := range flag.Value() {
				items[i] = item
			}
			value = items
		default:
			return nil, fmt.Errorf("unsupported prompt flag --%s", flag.Name())
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
	case "project", "app", "environment", "keyId", "apiId", "permission", "role", "roleId", "identity", "namespace", "portal", "deploymentId", "policyId":
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
