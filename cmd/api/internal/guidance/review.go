package guidance

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/unkeyed/unkey/pkg/prompt"
)

func (f *form) confirm(ctx context.Context, schema *Schema, values map[string]any, action string, needsChanges bool) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		selected, err := f.variant(schema, values, "")
		if err != nil {
			return err
		}
		if err := f.preview(selected, values, ""); err != nil {
			return err
		}
		choice, err := f.prompt.SelectOrdered("Continue?", []prompt.SelectOption{
			{Key: "confirm", Label: action},
			{Key: "back", Label: "Back"},
			{Key: "cancel", Label: "Cancel"},
		})
		if err != nil {
			return err
		}
		switch choice {
		case "confirm":
			return nil
		case "cancel":
			return context.Canceled
		case "back":
			if err := f.review(ctx, schema, values, "", needsChanges); err != nil {
				return err
			}
		}
	}
}

func (f *form) preview(schema *Schema, values map[string]any, path string) error {
	var preview strings.Builder
	writeReview(&preview, "", reviewValue(schema, values, path), "  ")
	_, err := fmt.Fprintf(f.out, "\n%s\n%s\n", reviewTitle(path), preview.String())
	return err
}

func (f *form) review(ctx context.Context, schema *Schema, values map[string]any, path string, needsChanges bool) error {
	configuration := schema
	schema, err := f.variant(schema, values, path)
	if err != nil {
		return err
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := f.preview(schema, values, path); err != nil {
			return err
		}
		options := []prompt.SelectOption{}
		if !missingRequired(schema, values) && (!needsChanges || hasChanges(schema, values)) {
			options = append(options, prompt.SelectOption{Key: "done", Label: "Continue"})
			if path != "" {
				options[0].Label = "Keep changes"
			}
		}
		for _, name := range orderedFields(schema) {
			if path == "" && slices.ContainsFunc(f.conflicts[name], func(other string) bool {
				_, present := values[other]
				return other != name && present
			}) {
				continue
			}
			verb := "Add "
			if _, present := values[name]; present {
				verb = "Edit "
			}
			options = append(options, prompt.SelectOption{Key: name, Label: verb + label(name)})
		}
		if len(configuration.Alternatives) > 0 {
			options = append(options, prompt.SelectOption{Key: "configuration", Label: "Change configuration"})
		}
		name, err := f.prompt.SelectOrdered("Next", options)
		if err != nil || name == "done" {
			return err
		}
		if name == "configuration" {
			replacement := map[string]any{}
			for name, value := range values {
				if configuration.Properties[name] != nil {
					replacement[name] = value
				}
			}
			for _, alternative := range configuration.Alternatives {
				for _, name := range alternative.Required {
					if !slices.Contains(configuration.Required, name) {
						delete(replacement, name)
					}
				}
			}
			if err := f.object(ctx, configuration, replacement, path, needsChanges); err != nil {
				return err
			}
			clear(values)
			maps.Copy(values, replacement)
			schema, err = f.variant(configuration, values, path)
			if err != nil {
				return err
			}
			continue
		}
		if _, present := values[name]; present && !slices.Contains(schema.Required, name) {
			choice, err := f.prompt.SelectOrdered(label(name), []prompt.SelectOption{
				{Key: "edit", Label: "Change value"},
				{Key: "omit", Label: "Omit from request"},
				{Key: "back", Label: "Back"},
			})
			if err != nil {
				return err
			}
			if choice == "omit" {
				delete(values, name)
				continue
			}
			if choice == "back" {
				continue
			}
		}
		if err := f.edit(ctx, schema.Properties[name], values, name, path); err != nil {
			return err
		}
	}
}

func (f *form) edit(ctx context.Context, schema *Schema, values map[string]any, name, path string) error {
	childPath := strings.TrimPrefix(path+"."+name, ".")
	_, emptyErr := scalarValue(schema, "")
	if schema.Type == "string" && emptyErr == nil && len(schema.Enum) == 0 {
		choice, err := f.prompt.SelectOrdered(label(childPath), []prompt.SelectOption{
			{Key: "edit", Label: "Enter text"}, {Key: "empty", Label: "Use empty text"},
		})
		if err != nil {
			return err
		}
		if choice == "empty" {
			values[name] = ""
			return nil
		}
	}
	if object, ok := values[name].(map[string]any); ok && (len(schema.Properties) > 0 || schema.Additional != nil) {
		choice, err := f.prompt.SelectOrdered(label(childPath), []prompt.SelectOption{
			{Key: "edit", Label: "Edit fields"}, {Key: "replace", Label: "Replace value"},
		})
		if err != nil {
			return err
		}
		if choice == "edit" {
			if len(schema.Properties) == 0 {
				return f.editDictionary(ctx, schema, object, childPath)
			}
			return f.review(ctx, schema, object, childPath, false)
		}
	}
	if items, ok := values[name].([]any); ok && schema.Type == "array" {
		updated, err := f.editList(ctx, schema, items, childPath)
		if err != nil {
			return err
		}
		values[name] = updated
		return nil
	}
	return f.field(ctx, schema, values, name, path)
}

func (f *form) editDictionary(ctx context.Context, schema *Schema, values map[string]any, path string) error {
	for {
		names := slices.Sorted(maps.Keys(values))
		options := []prompt.SelectOption{{Key: "done", Label: "Keep this object"}, {Key: "add", Label: "Add entry"}}
		for i, name := range names {
			options = append(options, prompt.SelectOption{Key: strconv.Itoa(i), Label: "Edit " + label(name)})
		}
		choice, err := f.prompt.SelectOrdered(label(path), options)
		if err != nil || choice == "done" {
			return err
		}
		if choice == "add" {
			name, err := f.prompt.String("Entry name")
			if err != nil {
				return err
			}
			if name == "" {
				continue
			}
			if err := f.field(ctx, schema.Additional, values, name, path); err != nil {
				return err
			}
			continue
		}
		index, err := strconv.Atoi(choice)
		if err != nil || index < 0 || index >= len(names) {
			return fmt.Errorf("invalid entry selection")
		}
		name := names[index]
		choice, err = f.prompt.SelectOrdered(label(name), []prompt.SelectOption{
			{Key: "edit", Label: "Change value"}, {Key: "remove", Label: "Remove entry"}, {Key: "back", Label: "Back"},
		})
		if err != nil {
			return err
		}
		if choice == "remove" {
			delete(values, name)
		} else if choice == "edit" {
			if err := f.edit(ctx, schema.Additional, values, name, path); err != nil {
				return err
			}
		}
	}
}

func (f *form) editList(ctx context.Context, schema *Schema, values []any, path string) ([]any, error) {
	for {
		options := []prompt.SelectOption{}
		if len(values) >= schema.MinItems {
			options = append(options, prompt.SelectOption{Key: "done", Label: "Keep this list"})
		}
		if schema.MaxItems == 0 || len(values) < schema.MaxItems {
			options = append(options, prompt.SelectOption{Key: "add", Label: "Add item"})
		}
		for i := range values {
			options = append(options, prompt.SelectOption{Key: strconv.Itoa(i), Label: fmt.Sprintf("Edit item %d", i+1)})
		}
		choice, err := f.prompt.SelectOrdered(label(path), options)
		if err != nil || choice == "done" {
			return values, err
		}
		if choice == "add" {
			value, err := f.value(ctx, schema.Items, fmt.Sprintf("%s[%d]", path, len(values)+1), "")
			if err != nil {
				return nil, err
			}
			values = append(values, value)
			continue
		}
		index, err := strconv.Atoi(choice)
		if err != nil || index < 0 || index >= len(values) {
			return nil, fmt.Errorf("invalid item selection")
		}
		choice, err = f.prompt.SelectOrdered("Item", []prompt.SelectOption{
			{Key: "edit", Label: "Change value"}, {Key: "remove", Label: "Remove item"}, {Key: "back", Label: "Back"},
		})
		if err != nil {
			return nil, err
		}
		if choice == "remove" {
			values = slices.Delete(values, index, index+1)
		} else if choice == "edit" {
			name := fmt.Sprintf("%s[%d]", path, index+1)
			item := map[string]any{name: values[index]}
			if err := f.edit(ctx, schema.Items, item, name, ""); err != nil {
				return nil, err
			}
			values[index] = item[name]
		}
	}
}

func reviewTitle(path string) string {
	if path == "" {
		return "Review request"
	}
	return "Review " + label(path)
}

func writeReview(output *strings.Builder, name string, value any, indent string) {
	prefix := indent + name
	if name != "" {
		prefix += ":"
	}
	switch value := value.(type) {
	case map[string]any:
		if len(value) == 0 {
			fmt.Fprintln(output, prefix, "(no fields)")
			return
		}
		if name != "" {
			fmt.Fprintln(output, prefix)
			indent += "  "
		}
		for _, key := range slices.Sorted(maps.Keys(value)) {
			writeReview(output, label(key), value[key], indent)
		}
	case []any:
		if len(value) == 0 {
			fmt.Fprintln(output, prefix, "(no items)")
			return
		}
		fmt.Fprintln(output, prefix)
		for i, item := range value {
			writeReview(output, fmt.Sprintf("Item %d", i+1), item, indent+"  ")
		}
	default:
		fmt.Fprintln(output, prefix, reviewScalar(value))
	}
}

func reviewScalar(value any) string {
	switch value := value.(type) {
	case nil:
		return "Clear (null)"
	case bool:
		if value {
			return "Yes"
		}
		return "No"
	case string:
		if value == "" {
			return "(empty text)"
		}
		if strings.TrimSpace(value) != value || strings.ContainsAny(value, "\n\r\t") {
			return terminalSafe(strconv.Quote(value))
		}
	}
	return terminalSafe(fmt.Sprint(value))
}

func reviewValue(schema *Schema, value any, path string) any {
	if schema == nil || schema.Secret || strings.HasSuffix(path, ".value") || schema.Type == "any" {
		return "[hidden]"
	}
	switch value := value.(type) {
	case map[string]any:
		properties := maps.Clone(schema.Properties)
		if properties == nil {
			properties = map[string]*Schema{}
		}
		for _, alternative := range schema.Alternatives {
			maps.Copy(properties, alternative.Properties)
		}
		result := make(map[string]any, len(value))
		for name, child := range value {
			result[name] = reviewValue(properties[name], child, path+"."+name)
		}
		return result
	case []any:
		result := make([]any, len(value))
		for i, child := range value {
			result[i] = reviewValue(schema.Items, child, fmt.Sprintf("%s[%d]", path, i+1))
		}
		return result
	default:
		return value
	}
}

func terminalSafe(value string) string {
	var result strings.Builder
	for _, char := range value {
		if unicode.IsControl(char) && char != '\n' || unicode.Is(unicode.Cf, char) {
			result.WriteString(strconv.QuoteRune(char))
		} else {
			result.WriteRune(char)
		}
	}
	return result.String()
}
