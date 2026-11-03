package guidance

import (
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/unkeyed/unkey/pkg/prompt"
)

var errReviewEdited = errors.New("edit before confirmation")

type reviewMode struct {
	action      string
	warning     string
	expanded    bool
	destructive bool
}

func (f *form) preview(schema *Schema, values map[string]any, path string) error {
	if f.screen != nil {
		return nil
	}
	_, err := io.WriteString(f.out, f.summary(palette{color: false, accent: ""}, schema, values, path))
	return err
}

// show keeps the request summary above the active prompt until the returned function runs.
func (f *form) show(schema *Schema, values map[string]any, path string) func() {
	return f.section(path, func(p palette) string { return f.summary(p, schema, values, path) })
}

func (f *form) section(path string, render func(palette) string) func() {
	if f.screen == nil || slices.ContainsFunc(f.screen.sections, func(s section) bool { return s.path == path }) {
		return func() {}
	}
	count := len(f.screen.sections)
	f.screen.sections = append(f.screen.sections, section{path: path, render: render})
	return func() { f.screen.sections = f.screen.sections[:count] }
}

func (f *form) review(ctx context.Context, schema *Schema, values map[string]any, path string, needsChanges bool) error {
	return f.reviewMenu(ctx, schema, values, path, needsChanges, reviewMode{action: "", warning: "", expanded: true, destructive: false})
}

func (f *form) reviewMenu(ctx context.Context, schema *Schema, values map[string]any, path string, needsChanges bool, mode reviewMode) error {
	configuration := schema
	schema, err := f.variant(schema, values, path)
	if err != nil {
		return err
	}
	if path != "" {
		defer f.show(configuration, values, path)()
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := f.preview(schema, values, path); err != nil {
			return err
		}
		if f.screen == nil && mode.warning != "" {
			if _, err := fmt.Fprintln(f.out, mode.warning); err != nil {
				return err
			}
		}
		options := []prompt.SelectOption{}
		if !missingRequired(schema, values) && (!needsChanges || hasChanges(schema, values)) {
			action := "Keep changes"
			if mode.action != "" {
				action = mode.action
			}
			options = append(options, prompt.SelectOption{Key: "done", Label: action})
		}
		fields := orderedFields(schema)
		if mode.action != "" {
			slices.SortStableFunc(fields, func(a, b string) int {
				if targetField(a) == targetField(b) {
					return 0
				}
				if targetField(a) {
					return 1
				}
				return -1
			})
		}
		if !mode.expanded {
			fields = nil
			options = append(options, prompt.SelectOption{Key: "expand", Label: "Edit…"})
		}
		for _, name := range fields {
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
			if mode.action != "" {
				verb = "Set "
				if _, present := values[name]; present {
					verb = "Change "
				}
			}
			options = append(options, prompt.SelectOption{Key: name, Label: verb + label(name)})
		}
		if mode.expanded && len(configuration.Alternatives) > 0 {
			options = append(options, prompt.SelectOption{Key: "configuration", Label: "Change configuration"})
		}
		question := "Edit fields"
		defaultValue := ""
		if mode.action != "" {
			question = ""
			options = append(options, prompt.SelectOption{Key: "cancel", Label: "Cancel"})
			if mode.destructive {
				defaultValue = "cancel"
			}
		}
		name, err := f.prompt.SelectOrdered(question, options, defaultValue)
		if err != nil || name == "done" {
			return err
		}
		if name == "cancel" {
			return context.Canceled
		}
		if name == "expand" {
			mode.expanded = true
			continue
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
			if mode.destructive {
				return errReviewEdited
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
				if mode.destructive {
					return errReviewEdited
				}
				continue
			}
			if choice == "back" {
				continue
			}
		}
		if err := f.edit(ctx, schema.Properties[name], values, name, path); err != nil {
			return err
		}
		if mode.destructive {
			return errReviewEdited
		}
	}
}

func (f *form) edit(ctx context.Context, schema *Schema, values map[string]any, name, path string) error {
	if f.screen != nil {
		sections := f.screen.sections
		f.screen.sections = slices.DeleteFunc(slices.Clone(sections), func(s section) bool { return s.path == "confirmation" })
		defer func() { f.screen.sections = sections }()
	}
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
	if _, ok := values[name].([]any); ok && schema.Type == "array" {
		return f.editList(ctx, schema, values, name, childPath)
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

func (f *form) editList(ctx context.Context, schema *Schema, fields map[string]any, name, path string) error {
	values, ok := fields[name].([]any)
	if !ok {
		return fmt.Errorf("invalid list value for %s", path)
	}
	for {
		fields[name] = values
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
			return err
		}
		if choice == "add" {
			value, err := f.value(ctx, schema.Items, fmt.Sprintf("%s[%d]", path, len(values)+1), "")
			if err != nil {
				return err
			}
			values = append(values, value)
			continue
		}
		index, err := strconv.Atoi(choice)
		if err != nil || index < 0 || index >= len(values) {
			return fmt.Errorf("invalid item selection")
		}
		choice, err = f.prompt.SelectOrdered("Item", []prompt.SelectOption{
			{Key: "edit", Label: "Change value"}, {Key: "remove", Label: "Remove item"}, {Key: "back", Label: "Back"},
		})
		if err != nil {
			return err
		}
		if choice == "remove" {
			values = slices.Delete(values, index, index+1)
		} else if choice == "edit" {
			name := fmt.Sprintf("%s[%d]", path, index+1)
			item := map[string]any{name: values[index]}
			if err := f.edit(ctx, schema.Items, item, name, ""); err != nil {
				return err
			}
			values[index] = item[name]
		}
	}
}

func (f *form) summary(p palette, schema *Schema, values map[string]any, path string) string {
	if path == "" && len(f.selections) > 0 {
		values = maps.Clone(values)
		for field, selection := range f.selections {
			if values[field] == strings.TrimPrefix(selection.Key, "id:") {
				values[field] = selection.Label
			}
		}
	}
	return summary(p, schema, values, path)
}

func summary(p palette, schema *Schema, values map[string]any, path string) string {
	fields, ok := reviewValue(schema, values, path).(map[string]any)
	if !ok || len(fields) == 0 {
		return ""
	}
	var output strings.Builder
	indent := ""
	if path != "" {
		output.WriteString(p.bold(label(path)) + "\n")
		indent = "  "
	}
	for _, name := range orderedNames(slices.Collect(maps.Keys(fields))) {
		writeSummary(&output, p, p.done("✓")+" "+p.bold(labelPart(name)), fields[name], indent, indent+"    ")
	}
	return output.String()
}

func writeSummary(output *strings.Builder, p palette, head string, value any, indent, children string) {
	switch value := value.(type) {
	case map[string]any:
		if len(value) == 0 {
			fmt.Fprintln(output, indent+head+p.separator()+p.hint("(no fields)"))
			return
		}
		fmt.Fprintln(output, indent+head+strings.TrimSpace(p.separator()))
		for _, key := range slices.Sorted(maps.Keys(value)) {
			writeSummary(output, p, labelPart(key), value[key], children, children+"  ")
		}
	case []any:
		if len(value) == 0 {
			fmt.Fprintln(output, indent+head+p.separator()+p.hint("(no items)"))
			return
		}
		fmt.Fprintln(output, indent+head+strings.TrimSpace(p.separator()))
		for i, item := range value {
			writeSummary(output, p, fmt.Sprintf("Item %d", i+1), item, children, children+"  ")
		}
	default:
		fmt.Fprintln(output, indent+head+p.separator()+p.value(reviewScalar(value)))
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
