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

type treeNode struct {
	text      string
	children  []treeNode
	unwrapped bool
}

func writeDetails(output *strings.Builder, value any, indent string, format tableFormat) {
	object, ok := value.(map[string]any)
	if !ok {
		fmt.Fprintln(output, indent+cell(value))
		return
	}
	renderer := tui.New(os.Stdout)
	object = maps.Clone(object)
	if name, ok := object["name"].(string); ok && name != "" {
		writeTreeLine(output, indent, indent, terminalText(name), format.width, renderer.Bold)
		delete(object, "name")
	}
	var dates []treeNode
	for _, field := range []string{"createdAt", "updatedAt", "lastUsedAt", "expires"} {
		if value, ok := object[field]; ok {
			dates = append(dates, treeField(field, value, false))
			delete(object, field)
		}
	}
	nodes := treeFields(object, false)
	if len(dates) > 0 {
		nodes = append(nodes, treeNode{text: "Dates", children: dates, unwrapped: false})
	}
	writeTreeNodes(output, nodes, indent, format.width, renderer)
}

func treeFields(object map[string]any, metadata bool) []treeNode {
	keys := slices.Sorted(maps.Keys(object))
	if !metadata {
		for _, first := range []string{"id", "keyId"} {
			if i := slices.Index(keys, first); i >= 0 {
				keys = append([]string{first}, append(keys[:i], keys[i+1:]...)...)
			}
		}
	}
	var fields, groups []treeNode
	for _, field := range keys {
		node := treeField(field, object[field], metadata)
		switch object[field].(type) {
		case map[string]any, []any:
			groups = append(groups, node)
		default:
			fields = append(fields, node)
		}
	}
	return append(fields, groups...)
}

func treeField(field string, value any, metadata bool) treeNode {
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
	node := treeNode{text: label, children: nil, unwrapped: !metadata && (field == "key" || field == "plaintext")}
	switch value := value.(type) {
	case map[string]any:
		if remaining, ok := value["remaining"]; !metadata && field == "credits" && ok && remaining == nil {
			value = maps.Clone(value)
			value["remaining"] = "unlimited"
		}
		node.children = treeFields(value, metadata || field == "meta")
	case []any:
		for i, item := range value {
			if text, ok := item.(string); ok && !metadata {
				node.children = append(node.children, treeNode{text: cell(text), children: nil, unwrapped: false})
				continue
			}
			if object, ok := item.(map[string]any); ok && !metadata {
				if name, ok := object["name"].(string); ok && name != "" {
					object = maps.Clone(object)
					delete(object, "name")
					node.children = append(node.children, treeNode{text: terminalText(name), children: treeFields(object, false), unwrapped: false})
					continue
				}
			}
			node.children = append(node.children, treeField(fmt.Sprintf("[%d]", i), item, metadata))
		}
	default:
		text := fieldValue(field, value)
		if metadata {
			text = cell(value)
		} else if field == "duration" {
			var format tableFormat
			text = tableValue(field, value, format)
		}
		node.text += ": " + text
		return node
	}
	if len(node.children) == 0 {
		node.text += ": none"
	}
	return node
}

func writeTreeNodes(output *strings.Builder, nodes []treeNode, prefix string, width int, renderer *tui.Renderer) {
	for i, node := range nodes {
		branch, continuation := "├─ ", "│  "
		if i == len(nodes)-1 {
			branch, continuation = "└─ ", "   "
		}
		style := func(text string) string { return text }
		if len(node.children) > 0 {
			style = renderer.Bold
		}
		lineWidth := width
		if node.unwrapped {
			lineWidth = 0
		}
		writeTreeLine(output, prefix+branch, prefix+continuation, node.text, lineWidth, style)
		writeTreeNodes(output, node.children, prefix+continuation, width, renderer)
	}
}

func writeTreeLine(output *strings.Builder, prefix, continuation, text string, width int, style func(string) string) {
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
