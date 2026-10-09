package jwt

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// normalizeAudience rewrites a JSON-string aud claim into a one-element array
// so [RegisteredClaims] can decode it. Array audiences are left unchanged.
func normalizeAudience(payload []byte) ([]byte, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil, fmt.Errorf("invalid payload JSON: %w", err)
	}
	aud, ok := raw["aud"]
	if !ok {
		return payload, nil
	}
	trimmed := bytes.TrimSpace(aud)
	if len(trimmed) == 0 || trimmed[0] != '"' {
		return payload, nil
	}
	var audience string
	if err := json.Unmarshal(trimmed, &audience); err != nil {
		return nil, fmt.Errorf("invalid payload JSON: %w", err)
	}
	encoded, err := json.Marshal([]string{audience})
	if err != nil {
		return nil, fmt.Errorf("invalid payload JSON: %w", err)
	}
	raw["aud"] = encoded
	normalized, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid payload JSON: %w", err)
	}
	return normalized, nil
}
