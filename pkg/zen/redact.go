package zen

import "regexp"

type redactionRule struct {
	regexp      *regexp.Regexp
	replacement []byte
}

var redactionRules = []redactionRule{
	// Redact credential-bearing JSON fields with various whitespace combinations.
	{
		regexp:      regexp.MustCompile(`"(key|plaintext|token|session|sessionId)"\s*:\s*"[^"\\]*(?:\\.[^"\\]*)*"`),
		replacement: []byte(`"$1": "[REDACTED]"`),
	},
}

func redact(in []byte) []byte {
	b := in

	for _, rule := range redactionRules {
		b = rule.regexp.ReplaceAll(b, rule.replacement)
	}

	return b
}
