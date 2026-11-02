package util

import (
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/rivo/uniseg"
	"github.com/unkeyed/unkey/pkg/tui"
)

type tableFormat struct {
	width int
	now   time.Time
}

func writeTable(output *strings.Builder, rows []any, columns []string, indent string, format tableFormat) error {
	renderer := tui.New(os.Stdout)
	curated := len(columns) > 0
	if !curated {
		fields := map[string]struct{}{}
		for _, row := range rows {
			object, ok := row.(map[string]any)
			if !ok {
				for _, value := range rows {
					fmt.Fprintln(output, indent+cell(value))
				}
				return nil
			}
			for key := range object {
				fields[key] = struct{}{}
			}
		}
		columns = slices.Sorted(maps.Keys(fields))
	}
	labels := make([]string, len(columns))
	widths := make([]int, len(columns))
	for i, column := range columns {
		labels[i] = fieldLabel(column)
		if curated {
			labels[i] = tableLabel(column)
		}
		if format.width == 0 {
			labels[i] = strings.ToUpper(labels[i])
		} else {
			words := strings.Fields(labels[i])
			for j, word := range words {
				letters := []rune(word)
				letters[0] = unicode.ToUpper(letters[0])
				words[j] = string(letters)
			}
			labels[i] = strings.Join(words, " ")
		}
		widths[i] = uniseg.StringWidth(labels[i])
	}
	values := make([][]string, len(rows))
	for i, row := range rows {
		object, ok := row.(map[string]any)
		if !ok {
			return fmt.Errorf("expected a table row, got %T", row)
		}
		values[i] = make([]string, len(columns))
		for j, column := range columns {
			value := cell(object[column])
			if curated {
				value = tableValue(column, object[column], format)
			}
			values[i][j] = value
			widths[j] = max(widths[j], uniseg.StringWidth(value))
		}
	}
	style := func(index int, value string) string {
		if !curated {
			return value
		}
		return styleTableCell(renderer, columns[index], value, index == 0)
	}
	if format.width > 0 && !fitTable(widths, columns, curated, format.width-uniseg.StringWidth(indent)) {
		for i, row := range values {
			if i > 0 {
				output.WriteByte('\n')
			}
			for j, value := range row {
				fmt.Fprintf(output, "%s%s %s\n", indent, renderer.Dim(labels[j]+":"), style(j, value))
			}
		}
		return nil
	}
	writeTableRow(output, labels, widths, indent, func(_ int, value string) string { return renderer.Dim(value) })
	for _, row := range values {
		writeTableRow(output, row, widths, indent, style)
	}
	return nil
}

func styleTableCell(renderer *tui.Renderer, column, value string, primary bool) string {
	switch column {
	case "createdAt", "updatedAt", "lastUsedAt", "expires", "time":
		return renderer.Dim(value)
	case "status", "enabled":
		switch strings.ToLower(value) {
		case "verified", "ready", "enabled":
			return renderer.Green(value)
		case "pending", "verifying":
			return renderer.Yellow(value)
		case "failed":
			return renderer.Red(value)
		}
	}
	if primary {
		return renderer.Bold(value)
	}
	return value
}

func fitTable(widths []int, columns []string, curated bool, width int) bool {
	excess := 2*(len(widths)-1) - width
	for _, size := range widths {
		excess += size
	}
	if excess <= 0 {
		return true
	}
	if !curated {
		return false
	}
	nameHasSelector := slices.Contains(columns, "slug") || slices.Contains(columns, "id") || slices.Contains(columns, "keyId")
	for i, column := range columns {
		if column != "description" && !(column == "name" && nameHasSelector) {
			continue
		}
		reduction := min(excess, max(0, widths[i]-12))
		widths[i] -= reduction
		excess -= reduction
		if excess == 0 {
			return true
		}
	}
	return false
}

func writeTableRow(output *strings.Builder, row []string, widths []int, indent string, style func(int, string) string) {
	output.WriteString(indent)
	for i, value := range row {
		value = shortenCell(value, widths[i])
		output.WriteString(style(i, value))
		if i < len(row)-1 {
			output.WriteString(strings.Repeat(" ", widths[i]-uniseg.StringWidth(value)+2))
		}
	}
	output.WriteByte('\n')
}

func shortenCell(value string, width int) string {
	if uniseg.StringWidth(value) <= width {
		return value
	}
	var output strings.Builder
	graphemes := uniseg.NewGraphemes(value)
	used := 0
	for graphemes.Next() {
		if used+graphemes.Width() > width-1 {
			break
		}
		output.WriteString(graphemes.Str())
		used += graphemes.Width()
	}
	output.WriteRune('…')
	return output.String()
}

func tableLabel(key string) string {
	switch key {
	case "createdAt":
		return "Created"
	case "updatedAt":
		return "Updated"
	case "lastUsedAt":
		return "Last used"
	case "sourceType":
		return "Source"
	case "enabled":
		return "Status"
	case "isCurrent":
		return "Current"
	default:
		return fieldLabel(key)
	}
}
