package util

import (
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/rivo/uniseg"
	"github.com/unkeyed/unkey/pkg/tui"
)

type keyTreeNode struct {
	text     string
	children []keyTreeNode
}

func writeKeyTree(output *strings.Builder, value any, format tableFormat) error {
	key, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("expected key details, got %T", value)
	}
	renderer := tui.New(os.Stdout)
	name := "API key"
	if value, ok := key["name"].(string); ok && value != "" {
		name = terminalText(value)
	}
	status := "Disabled"
	if key["enabled"] == true {
		status = "Enabled"
	}
	plain := func(text string) string { return text }
	writeKeyTreeLine(output, "", "", name, format.width, renderer.Bold)
	writeKeyTreeLine(output, "", "", status+" · "+cell(key["keyId"]), format.width, plain)
	writeKeyTreeLine(output, "", "", "Prefix: "+cell(key["start"]), format.width, plain)
	output.WriteByte('\n')
	for _, field := range []string{"name", "enabled", "keyId", "start"} {
		delete(key, field)
	}

	var dates []keyTreeNode
	for _, field := range []string{"createdAt", "updatedAt", "lastUsedAt", "expires"} {
		if value, ok := key[field]; ok {
			dates = append(dates, keyTreeField(field, value, false))
			delete(key, field)
		}
	}
	var nodes []keyTreeNode
	for _, field := range []string{"identity", "permissions", "roles", "credits", "ratelimits", "meta", "plaintext"} {
		if value, ok := key[field]; ok {
			nodes = append(nodes, keyTreeField(field, value, false))
			delete(key, field)
		}
	}
	for _, field := range slices.Sorted(maps.Keys(key)) {
		nodes = append(nodes, keyTreeField(field, key[field], false))
	}
	if len(dates) > 0 {
		nodes = append(nodes, keyTreeNode{text: "Dates", children: dates})
	}
	writeKeyTreeNodes(output, nodes, "", format.width, renderer)
	return nil
}

func keyTreeField(field string, value any, metadata bool) keyTreeNode {
	label := fieldLabel(field)
	if metadata {
		label = terminalText(field)
	} else {
		switch field {
		case "meta":
			label = "Metadata"
		case "ratelimits":
			label = "Rate limits"
		}
	}
	node := keyTreeNode{text: label, children: nil}
	switch value := value.(type) {
	case map[string]any:
		for _, child := range slices.Sorted(maps.Keys(value)) {
			if !metadata && field == "credits" && child == "remaining" && value[child] == nil {
				node.children = append(node.children, keyTreeNode{text: "Remaining: unlimited", children: nil})
			} else {
				node.children = append(node.children, keyTreeField(child, value[child], metadata || field == "meta"))
			}
		}
	case []any:
		for i, item := range value {
			if rate, ok := item.(map[string]any); ok && !metadata && field == "ratelimits" {
				var format tableFormat
				node.children = append(node.children, keyTreeNode{
					text: cell(rate["name"]) + ": " + cell(rate["limit"]) + " / " + tableValue("duration", rate["duration"], format),
					children: []keyTreeNode{
						{text: "ID: " + cell(rate["id"]), children: nil},
						{text: "Auto apply: " + cell(rate["autoApply"]), children: nil},
					},
				})
			} else if text, ok := item.(string); ok && !metadata {
				node.children = append(node.children, keyTreeNode{text: cell(text), children: nil})
			} else {
				node.children = append(node.children, keyTreeField(fmt.Sprintf("[%d]", i), item, metadata))
			}
		}
	default:
		text := fieldValue(field, value)
		if metadata {
			text = cell(value)
		}
		node.text += ": " + text
		return node
	}
	if len(node.children) == 0 {
		node.text += ": none"
	}
	return node
}

func writeKeyTreeNodes(output *strings.Builder, nodes []keyTreeNode, prefix string, width int, renderer *tui.Renderer) {
	for i, node := range nodes {
		branch, continuation := "├─ ", "│  "
		if i == len(nodes)-1 {
			branch, continuation = "└─ ", "   "
		}
		style := func(text string) string { return text }
		if len(node.children) > 0 {
			style = renderer.Bold
		}
		writeKeyTreeLine(output, prefix+branch, prefix+continuation, node.text, width, style)
		writeKeyTreeNodes(output, node.children, prefix+continuation, width, renderer)
	}
}

func writeKeyTreeLine(output *strings.Builder, prefix, continuation, text string, width int, style func(string) string) {
	available := width - uniseg.StringWidth(prefix)
	if width <= 0 || available < 2 {
		fmt.Fprintln(output, prefix+style(text))
		return
	}
	var line strings.Builder
	lineWidth := 0
	graphemes := uniseg.NewGraphemes(text)
	for graphemes.Next() {
		if lineWidth+graphemes.Width() > available && lineWidth > 0 {
			fmt.Fprintln(output, prefix+style(line.String()))
			prefix = continuation
			line.Reset()
			lineWidth = 0
		}
		line.WriteString(graphemes.Str())
		lineWidth += graphemes.Width()
	}
	fmt.Fprintln(output, prefix+style(line.String()))
}
