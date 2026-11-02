package guidance

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/unkeyed/unkey/pkg/prompt"
)

type questions interface {
	String(string, ...string) (string, error)
	Secret(string) (string, error)
	SelectOrdered(string, []prompt.SelectOption, ...string) (string, error)
}

type form struct {
	prompt    questions
	out       io.Writer
	screen    *terminalForm
	pick      func(context.Context, *form, string, map[string]any) (string, bool, error)
	conflicts map[string][]string
}

func (f *form) object(ctx context.Context, schema *Schema, values map[string]any, path string, needsChanges bool) error {
	schema, err := f.variant(schema, values, path)
	if err != nil {
		return err
	}
	if path != "" {
		defer f.show(schema, values, path)()
	}
	for _, name := range orderedFields(schema) {
		if !slices.Contains(schema.Required, name) {
			continue
		}
		if value, ok := values[name]; ok && value != "" {
			continue
		}
		if err := f.field(ctx, schema.Properties[name], values, name, path); err != nil {
			return err
		}
	}
	for {
		options := []prompt.SelectOption{}
		if !needsChanges || hasChanges(schema, values) {
			options = append(options, prompt.SelectOption{Key: "done", Label: "Continue"})
		}
		for _, name := range orderedFields(schema) {
			if path == "" && slices.ContainsFunc(f.conflicts[name], func(other string) bool {
				_, present := values[other]
				return present
			}) {
				continue
			}
			if _, present := values[name]; !present {
				options = append(options, prompt.SelectOption{Key: name, Label: label(name)})
			}
		}
		if len(options) == 0 || (len(options) == 1 && options[0].Key == "done") {
			return nil
		}
		question := "Additional settings"
		if needsChanges {
			question = "Choose a field to update"
		}
		if path != "" {
			question = label(path) + " · " + question
		}
		name, err := f.prompt.SelectOrdered(question, options)
		if err != nil {
			return err
		}
		if name == "done" {
			return nil
		}
		if err := f.field(ctx, schema.Properties[name], values, name, path); err != nil {
			return err
		}
	}
}

func (f *form) field(ctx context.Context, schema *Schema, values map[string]any, name, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if schema == nil {
		return fmt.Errorf("missing input definition for %s", name)
	}
	if path == "" && f.pick != nil && targetField(name) {
		value, handled, err := f.pick(ctx, f, name, values)
		if err != nil {
			return err
		}
		if handled {
			values[name] = value
			return nil
		}
	}
	path = strings.TrimPrefix(path+"."+name, ".")
	if schema.Nullable {
		choice, err := f.prompt.SelectOrdered(label(path), []prompt.SelectOption{
			{Key: "set", Label: "Set a value"},
			{Key: "clear", Label: "Clear (null)"},
		})
		if err != nil {
			return err
		}
		if choice == "clear" {
			values[name] = nil
			return nil
		}
	}
	defaultValue := ""
	if previous, present := values[name]; present && previous != nil && !schema.Secret && !strings.HasSuffix(path, ".value") {
		switch previous.(type) {
		case string, bool, json.Number, int, int64, float64:
			text := fmt.Sprint(previous)
			if terminalSafe(text) == text && !strings.ContainsAny(text, "\r\n") {
				defaultValue = text
			}
		}
	}
	if name == "slug" && defaultValue == "" {
		for _, source := range []string{"name", "displayName"} {
			if value, ok := values[source].(string); ok {
				defaultValue = slug(value)
				break
			}
		}
	}
	value, err := f.value(ctx, schema, path, defaultValue)
	if err != nil {
		return err
	}
	values[name] = value
	return nil
}

func (f *form) value(ctx context.Context, schema *Schema, path, defaultValue string) (any, error) {
	if defaultValue == "" && len(schema.Default) > 0 {
		value, err := decodeValue(schema.Default)
		if err != nil {
			return nil, err
		}
		defaultValue = fmt.Sprint(value)
	}
	if len(schema.Alternatives) > 0 && schema.Type != "object" {
		options := make([]prompt.SelectOption, len(schema.Alternatives))
		for i, alternative := range schema.Alternatives {
			options[i] = prompt.SelectOption{Key: strconv.Itoa(i), Label: variantLabel(alternative)}
		}
		choice, err := f.prompt.SelectOrdered(label(path)+" · Choose value type", options)
		if err != nil {
			return nil, err
		}
		index, err := strconv.Atoi(choice)
		if err != nil || index < 0 || index >= len(schema.Alternatives) {
			return nil, fmt.Errorf("invalid input type selection")
		}
		return f.value(ctx, schema.Alternatives[index], path, defaultValue)
	}
	if len(schema.Enum) > 0 {
		options := make([]prompt.SelectOption, len(schema.Enum))
		for i, value := range schema.Enum {
			options[i] = prompt.SelectOption{Key: value, Label: value}
		}
		return f.prompt.SelectOrdered(label(path), options, defaultValue)
	}
	switch schema.Type {
	case "object":
		values := map[string]any{}
		if len(schema.Properties) > 0 {
			return values, f.object(ctx, schema, values, path, false)
		}
		return f.dictionary(ctx, schema, path)
	case "array":
		return f.array(ctx, schema, path)
	case "boolean":
		choice, err := f.prompt.SelectOrdered(label(path), []prompt.SelectOption{
			{Key: "true", Label: "Yes"}, {Key: "false", Label: "No"},
		}, defaultValue)
		return choice == "true", err
	case "any", "":
		choice, err := f.prompt.SelectOrdered(label(path)+" · Value type", []prompt.SelectOption{
			{Key: "string", Label: "Text"}, {Key: "number", Label: "Number"},
			{Key: "boolean", Label: "Yes / No"}, {Key: "object", Label: "Object"},
			{Key: "array", Label: "List"}, {Key: "null", Label: "Null"},
		})
		if err != nil || choice == "null" {
			return nil, err
		}
		var arbitrary, selected Schema
		arbitrary.Type = "any"
		selected.Type = choice
		selected.Additional = &arbitrary
		selected.Items = &arbitrary
		return f.value(ctx, &selected, path, "")
	case "null":
		return nil, nil
	default:
		return f.scalar(schema, path, defaultValue)
	}
}

func (f *form) scalar(schema *Schema, path, defaultValue string) (any, error) {
	for {
		var text string
		var err error
		if schema.Secret || strings.HasSuffix(path, ".value") {
			text, err = f.prompt.Secret(label(path))
		} else {
			text, err = f.prompt.String(label(path), defaultValue)
		}
		if err != nil {
			return nil, err
		}
		value, err := scalarValue(schema, text)
		if err == nil {
			return value, nil
		}
		message := []rune(err.Error())
		message[0] = unicode.ToUpper(message[0])
		if _, err := fmt.Fprintln(f.out, string(message)); err != nil {
			return nil, err
		}
	}
}

func scalarValue(schema *Schema, text string) (any, error) {
	if schema.Type == "integer" || schema.Type == "number" {
		number, err := strconv.ParseFloat(text, 64)
		if !json.Valid([]byte(text)) || err != nil {
			return nil, fmt.Errorf("enter a %s", schema.Type)
		}
		if schema.Type == "integer" {
			if _, err := strconv.ParseInt(text, 10, 64); err != nil {
				return nil, fmt.Errorf("enter a whole number")
			}
		}
		if schema.Minimum != nil && number < *schema.Minimum {
			return nil, fmt.Errorf("must be at least %g", *schema.Minimum)
		}
		if schema.Maximum != nil && number > *schema.Maximum {
			return nil, fmt.Errorf("must be at most %g", *schema.Maximum)
		}
		return json.Number(text), nil
	}
	length := utf8.RuneCountInString(text)
	if length < schema.MinLength {
		if schema.MinLength == 1 {
			return nil, fmt.Errorf("enter a value")
		}
		return nil, fmt.Errorf("enter at least %d characters", schema.MinLength)
	}
	if schema.MaxLength > 0 && length > schema.MaxLength {
		return nil, fmt.Errorf("use at most %d characters", schema.MaxLength)
	}
	if schema.Pattern != "" {
		match, err := regexp.MatchString(schema.Pattern, text)
		if err != nil {
			return nil, err
		}
		if !match {
			return nil, fmt.Errorf("must match %s", schema.Pattern)
		}
	}
	return text, nil
}

func (f *form) array(ctx context.Context, schema *Schema, path string) ([]any, error) {
	values := []any{}
	name := path[strings.LastIndex(path, ".")+1:]
	var wrapper Schema
	wrapper.Type = "object"
	wrapper.Properties = map[string]*Schema{name: schema}
	defer f.section(path, func(p palette) string {
		if len(values) == 0 {
			return ""
		}
		return summary(p, &wrapper, map[string]any{name: values}, "")
	})()
	for schema.MaxItems == 0 || len(values) < schema.MaxItems {
		options := []prompt.SelectOption{{Key: "add", Label: "Add item"}}
		if len(values) >= schema.MinItems {
			options = append(options, prompt.SelectOption{Key: "done", Label: "Use this list (" + quantity(len(values), "item", "items") + ")"})
		}
		choice, err := f.prompt.SelectOrdered(label(path), options)
		if err != nil || choice == "done" {
			return values, err
		}
		value, err := f.value(ctx, schema.Items, fmt.Sprintf("%s[%d]", path, len(values)+1), "")
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

func (f *form) dictionary(ctx context.Context, schema *Schema, path string) (map[string]any, error) {
	values := map[string]any{}
	if schema.Additional == nil {
		return values, nil
	}
	for {
		choice, err := f.prompt.SelectOrdered(label(path), []prompt.SelectOption{
			{Key: "add", Label: "Add entry"}, {Key: "done", Label: "Use this object (" + quantity(len(values), "entry", "entries") + ")"},
		})
		if err != nil || choice == "done" {
			return values, err
		}
		name, err := f.prompt.String("Entry name")
		if err != nil {
			return nil, err
		}
		if name == "" {
			continue
		}
		value, err := f.value(ctx, schema.Additional, path+"."+name, "")
		if err != nil {
			return nil, err
		}
		values[name] = value
	}
}

func (f *form) variant(schema *Schema, values map[string]any, path string) (*Schema, error) {
	if len(schema.Alternatives) == 0 {
		return schema, nil
	}
	index := -1
	options := make([]prompt.SelectOption, len(schema.Alternatives))
	for i, alternative := range schema.Alternatives {
		options[i] = prompt.SelectOption{Key: strconv.Itoa(i), Label: variantLabel(alternative)}
		if !missingRequired(alternative, values) {
			index = i
		}
	}
	if index == -1 {
		choice, err := f.prompt.SelectOrdered(label(path)+" · Choose configuration", options)
		if err != nil {
			return nil, err
		}
		index, err = strconv.Atoi(choice)
		if err != nil || index < 0 || index >= len(schema.Alternatives) {
			return nil, fmt.Errorf("invalid configuration selection")
		}
	}
	selected := schema.Alternatives[index]
	merged := *schema
	merged.Properties = maps.Clone(schema.Properties)
	if merged.Properties == nil {
		merged.Properties = map[string]*Schema{}
	}
	maps.Copy(merged.Properties, selected.Properties)
	merged.Required = append(slices.Clone(schema.Required), selected.Required...)
	merged.Alternatives = nil
	for i, alternative := range schema.Alternatives {
		if i == index {
			continue
		}
		for _, name := range alternative.Required {
			if !slices.Contains(merged.Required, name) {
				delete(merged.Properties, name)
			}
		}
	}
	return &merged, nil
}

func variantLabel(schema *Schema) string {
	if len(schema.Required) > 0 {
		return label(strings.Join(schema.Required, ", "))
	}
	return label(schema.Type)
}

func orderedFields(schema *Schema) []string {
	return orderedNames(slices.Collect(maps.Keys(schema.Properties)))
}

func orderedNames(names []string) []string {
	slices.Sort(names)
	priority := []string{"project", "app", "environment", "apiId", "keyId", "name", "displayName", "slug"}
	slices.SortStableFunc(names, func(a, b string) int {
		indexA, indexB := slices.Index(priority, a), slices.Index(priority, b)
		if indexA < 0 {
			indexA = len(priority)
		}
		if indexB < 0 {
			indexB = len(priority)
		}
		return indexA - indexB
	})
	return names
}

func label(path string) string {
	parts := strings.Split(path, ".")
	for i, part := range parts {
		parts[i] = labelPart(part)
	}
	return strings.Join(parts, " › ")
}

func labelPart(name string) string {
	name, index, indexed := strings.Cut(name, "[")
	var text strings.Builder
	for i, char := range name {
		if unicode.IsControl(char) || unicode.Is(unicode.Cf, char) {
			text.WriteString(strconv.QuoteRune(char))
			continue
		}
		if i == 0 {
			text.WriteRune(unicode.ToUpper(char))
			continue
		}
		if unicode.IsUpper(char) {
			text.WriteByte(' ')
		}
		text.WriteRune(unicode.ToLower(char))
	}
	words := strings.Split(text.String(), " ")
	for i, word := range words {
		switch upper := strings.ToUpper(word); upper {
		case "API", "DNS", "HTTP", "ID", "IP", "JSON", "OCI", "TTL", "URL":
			words[i] = upper
		}
	}
	if indexed {
		words = append(words, "#"+labelPart(strings.TrimSuffix(index, "]")))
	}
	return strings.Join(words, " ")
}

func quantity(count int, singular, plural string) string {
	if count == 1 {
		return "1 " + singular
	}
	return strconv.Itoa(count) + " " + plural
}

func slug(name string) string {
	var text strings.Builder
	separator := false
	for _, char := range strings.ToLower(name) {
		if char >= 'a' && char <= 'z' || char >= '0' && char <= '9' {
			if separator && text.Len() > 0 {
				text.WriteByte('-')
			}
			text.WriteRune(char)
			separator = false
		} else {
			separator = true
		}
	}
	return text.String()
}
