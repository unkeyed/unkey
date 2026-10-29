package util

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"
	"unicode"
)

func writeDetails(output *strings.Builder, value any, indent string, metadata bool) error {
	object, ok := value.(map[string]any)
	if !ok {
		fmt.Fprintln(output, indent+cell(value))
		return nil
	}
	keys := slices.Sorted(maps.Keys(object))
	table := tabwriter.NewWriter(output, 0, 4, 2, ' ', 0)
	for _, key := range keys {
		switch object[key].(type) {
		case map[string]any, []any:
			continue
		default:
			label, value := fieldLabel(key), fieldValue(key, object[key])
			if metadata {
				label, value = terminalText(key), cell(object[key])
			}
			fmt.Fprintf(table, "%s%s:\t%s\n", indent, label, value)
		}
	}
	if err := table.Flush(); err != nil {
		return err
	}
	for _, key := range keys {
		switch nested := object[key].(type) {
		case map[string]any:
			if len(nested) == 0 {
				fmt.Fprintf(output, "%s%s: none\n", indent, fieldLabel(key))
				continue
			}
			fmt.Fprintf(output, "%s%s:\n", indent, fieldLabel(key))
			if err := writeDetails(output, nested, indent+"  ", metadata || key == "meta"); err != nil {
				return err
			}
		case []any:
			if len(nested) == 0 {
				fmt.Fprintf(output, "%s%s: none\n", indent, fieldLabel(key))
				continue
			}
			fmt.Fprintf(output, "%s%s:\n", indent, fieldLabel(key))
			if err := writeTable(output, nested, nil, indent+"  ", tableFormat{width: 0, now: time.Time{}}); err != nil {
				return err
			}
		}
	}
	return nil
}

func fieldValue(key string, value any) string {
	switch key {
	case "createdAt", "updatedAt", "lastUsedAt", "expires", "time":
		if number, ok := value.(json.Number); ok {
			if milliseconds, err := number.Int64(); err == nil && milliseconds > 0 {
				return time.UnixMilli(milliseconds).UTC().Format(time.RFC3339)
			}
		}
	}
	return cell(value)
}

func tableValue(key string, value any, format tableFormat) string {
	switch key {
	case "time", "expires":
		return fieldValue(key, value)
	case "createdAt", "updatedAt", "lastUsedAt":
		if value == nil {
			return "-"
		}
		if number, ok := value.(json.Number); ok {
			if milliseconds, err := number.Int64(); err == nil {
				if milliseconds <= 0 {
					return "-"
				}
				if format.width > 0 {
					return relativeTime(time.UnixMilli(milliseconds), format.now)
				}
				return fieldValue(key, value)
			}
		}
	case "enabled":
		if enabled, ok := value.(bool); ok {
			if enabled {
				return "Enabled"
			}
			return "Disabled"
		}
	case "isCurrent":
		if current, ok := value.(bool); ok {
			if current {
				return "Yes"
			}
			return "No"
		}
	case "duration":
		if number, ok := value.(json.Number); ok {
			if milliseconds, err := number.Int64(); err == nil && milliseconds >= 0 && milliseconds <= (1<<63-1)/int64(time.Millisecond) {
				return (time.Duration(milliseconds) * time.Millisecond).String()
			}
		}
	}
	return cell(value)
}

func relativeTime(timestamp, now time.Time) string {
	delta := now.Sub(timestamp)
	future := timestamp.After(now)
	if future {
		delta = timestamp.Sub(now)
	}
	if delta < time.Minute {
		if future {
			return "in a moment"
		}
		return "just now"
	}
	for _, unit := range []struct {
		duration time.Duration
		name     string
	}{
		{365 * 24 * time.Hour, "year"},
		{30 * 24 * time.Hour, "month"},
		{24 * time.Hour, "day"},
		{time.Hour, "hour"},
		{time.Minute, "minute"},
	} {
		if count := int64(delta / unit.duration); count > 0 {
			label := unit.name
			if count != 1 {
				label += "s"
			}
			if future {
				return fmt.Sprintf("in %d %s", count, label)
			}
			return fmt.Sprintf("%d %s ago", count, label)
		}
	}
	return "just now"
}

func cell(value any) string {
	switch value := value.(type) {
	case nil:
		return "-"
	case string:
		if value == "" {
			return "-"
		}
		return terminalText(value)
	case []any:
		if len(value) == 0 {
			return "none"
		}
		parts := make([]string, len(value))
		for i, item := range value {
			parts[i] = cell(item)
		}
		return strings.Join(parts, ", ")
	case map[string]any:
		if len(value) == 0 {
			return "none"
		}
		var parts []string
		for _, key := range slices.Sorted(maps.Keys(value)) {
			parts = append(parts, terminalText(key)+"="+cell(value[key]))
		}
		return strings.Join(parts, ", ")
	default:
		return fmt.Sprint(value)
	}
}

func fieldLabel(key string) string {
	var words strings.Builder
	for i, r := range key {
		if unicode.IsUpper(r) && i > 0 {
			words.WriteByte(' ')
		}
		if r == '_' || r == '-' {
			words.WriteByte(' ')
		} else {
			words.WriteRune(unicode.ToLower(r))
		}
	}
	label := strings.Fields(words.String())
	for i, word := range label {
		switch word {
		case "id", "url", "api", "dns", "ttl", "cpu":
			label[i] = strings.ToUpper(word)
		default:
			if i == 0 {
				letters := []rune(word)
				letters[0] = unicode.ToUpper(letters[0])
				label[i] = string(letters)
			}
		}
	}
	return terminalText(strings.Join(label, " "))
}

func terminalText(value string) string {
	var output strings.Builder
	for _, r := range value {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			escaped := strconv.QuoteRune(r)
			output.WriteString(escaped[1 : len(escaped)-1])
		} else {
			output.WriteRune(r)
		}
	}
	return output.String()
}
