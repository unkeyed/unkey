package util

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/rivo/uniseg"
)

type tableFormat struct {
	width int
	now   time.Time
}

func writeTable(output *strings.Builder, rows []any, columns []string, indent string, format tableFormat) error {
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
		widths[i] = uniseg.StringWidth(strings.ToUpper(labels[i]))
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
	if format.width > 0 && !fitTable(widths, columns, curated, format.width-uniseg.StringWidth(indent)) {
		for i, row := range values {
			if i > 0 {
				output.WriteByte('\n')
			}
			for j, value := range row {
				fmt.Fprintf(output, "%s%s: %s\n", indent, labels[j], value)
			}
		}
		return nil
	}
	for i := range labels {
		labels[i] = strings.ToUpper(labels[i])
	}
	writeTableRow(output, labels, widths, indent)
	for _, row := range values {
		writeTableRow(output, row, widths, indent)
	}
	return nil
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

func writeTableRow(output *strings.Builder, row []string, widths []int, indent string) {
	output.WriteString(indent)
	for i, value := range row {
		value = shortenCell(value, widths[i])
		output.WriteString(value)
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
