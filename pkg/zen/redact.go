package zen

import "regexp"

type redactionRule struct {
	regexp      *regexp.Regexp
	replacement []byte
}

var redactionRules = []redactionRule{
	// Redact "key" field values - matches JSON-style key fields with various whitespace combinations
	{
		regexp:      regexp.MustCompile(`"key"\s*:\s*"[^"\\]*(?:\\.[^"\\]*)*"`),
		replacement: []byte(`"key": "[REDACTED]"`),
	},
	// Redact "plaintext" field values - matches JSON-style plaintext fields with various whitespace combinations
	{
		regexp:      regexp.MustCompile(`"plaintext"\s*:\s*"[^"\\]*(?:\\.[^"\\]*)*"`),
		replacement: []byte(`"plaintext": "[REDACTED]"`),
	},
}

func redact(in []byte) []byte {
	b := in

	for _, rule := range redactionRules {
		b = rule.regexp.ReplaceAll(b, rule.replacement)
	}

	return b
}
