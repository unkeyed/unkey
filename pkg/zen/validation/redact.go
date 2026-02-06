package validation

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

// RedactionNode mirrors the schema tree. Only fields with an
// exact path match are redacted — "data.key" won't touch "metadata.key".
type RedactionNode struct {
	Redact   bool                      // true = replace value with [REDACTED]
	Children map[string]*RedactionNode // object properties
	Items    *RedactionNode            // array items
}

// RedactionConfig holds separate redaction trees for request and response bodies,
// plus per-route header names to redact.
type RedactionConfig struct {
	Request         *RedactionNode
	Response        *RedactionNode
	RedactedHeaders map[string]bool // lowercase header names to redact for this route
}

// RedactJSON applies path-aware redaction and returns compact JSON.
// Returns compact input unchanged if routeKey has no rules or input is not valid JSON.
func RedactJSON(redactions map[string]*RedactionConfig, routeKey string, isResponse bool, data []byte) []byte {
	config := redactions[routeKey]
	if config == nil {
		return CompactJSON(data)
	}
	node := config.Request
	if isResponse {
		node = config.Response
	}
	if node == nil {
		return CompactJSON(data)
	}
	var v any
	if json.Unmarshal(data, &v) != nil {
		return data
	}
	redactWalk(v, node)
	out, err := json.Marshal(v) // compact by default
	if err != nil {
		return data
	}
	return out
}

// CompactJSON returns compact JSON. If data is not valid JSON, it is returned as-is.
func CompactJSON(data []byte) []byte {
	var buf bytes.Buffer
	if json.Compact(&buf, data) == nil {
		return buf.Bytes()
	}
	return data
}

func redactWalk(v any, node *RedactionNode) {
	switch val := v.(type) {
	case map[string]any:
		for k, child := range val {
			childNode, ok := node.Children[k]
			if !ok {
				continue
			}
			if childNode.Redact {
				val[k] = "[REDACTED]"
			} else {
				redactWalk(child, childNode)
			}
		}
	case []any:
		if node.Items != nil {
			for _, item := range val {
				redactWalk(item, node.Items)
			}
		}
	}
}

// infraSkipHeaders are infrastructure headers filtered out entirely from logs.
var infraSkipHeaders = map[string]bool{
	"x-forwarded-proto": true,
	"x-forwarded-port":  true,
	"x-forwarded-for":   true,
	"x-amzn-trace-id":   true,
}

// FormatHeader formats a single header as "Key: Value".
func FormatHeader(key, value string) string {
	var b strings.Builder
	b.Grow(len(key) + 2 + len(value))
	b.WriteString(key)
	b.WriteString(": ")
	b.WriteString(value)
	return b.String()
}

// SanitizeHeaders formats HTTP headers as "Key: Value" strings, filtering out
// infrastructure headers and redacting values for headers in the redactSet.
func SanitizeHeaders(headers http.Header, redactSet map[string]bool) []string {
	result := make([]string, 0, len(headers))
	for k, vv := range headers {
		lk := strings.ToLower(k)
		if infraSkipHeaders[lk] {
			continue
		}
		value := strings.Join(vv, ",")
		if redactSet[lk] {
			value = "[REDACTED]"
		}
		result = append(result, FormatHeader(k, value))
	}
	return result
}

// NewRedactionTree builds a RedactionNode from a list of path segments.
// e.g., [["key"], ["data", "plaintext"]] → tree with key→redact, data→plaintext→redact
// The "[]" segment marks an array — it sets Items and continues into it.
func NewRedactionTree(paths [][]string) *RedactionNode {
	root := &RedactionNode{Children: map[string]*RedactionNode{}}
	for _, path := range paths {
		cur := root

		for i, seg := range path {
			if seg == "[]" {
				if cur.Items == nil {
					cur.Items = &RedactionNode{Children: map[string]*RedactionNode{}}
				}

				cur = cur.Items
				continue
			}

			if _, ok := cur.Children[seg]; !ok {
				cur.Children[seg] = &RedactionNode{Children: map[string]*RedactionNode{}}
			}

			if i == len(path)-1 {
				cur.Children[seg].Redact = true
			} else {
				cur = cur.Children[seg]
			}
		}
	}
	return root
}
