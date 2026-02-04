package wide

import (
	"strings"
)

// Note: Using strings.Builder for efficient string concatenation in loops

// MaskEmail masks an email address for safe logging.
// Example: "john.doe@example.com" -> "j***.d**@e******.com"
//
// Returns the original string if it's not a valid email format.
func MaskEmail(email string) string {
	if email == "" {
		return ""
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return email
	}

	local := parts[0]
	domain := parts[1]

	maskedLocal := maskString(local)
	maskedDomain := maskDomain(domain)

	return maskedLocal + "@" + maskedDomain
}

// MaskAPIKey masks an API key for safe logging, showing only prefix and last 4 chars.
// Example: "sk_live_abc123xyz789" -> "sk_live_***789"
// Example: "key_abc123xyz" -> "key_***xyz"
//
// If the key is too short (< 8 chars), returns "***".
func MaskAPIKey(key string) string {
	if len(key) < 8 {
		return "***"
	}

	// Find prefix (e.g., "sk_live_", "key_", "pk_")
	prefixEnd := 0
	underscoreCount := 0
	for i, c := range key {
		if c == '_' {
			underscoreCount++
			prefixEnd = i + 1
			// Stop after 2 underscores (e.g., "sk_live_") or if we've gone too far
			if underscoreCount >= 2 || i > 10 {
				break
			}
		}
	}

	// If no prefix found, just show last 4
	if prefixEnd == 0 || prefixEnd >= len(key)-4 {
		return "***" + key[len(key)-4:]
	}

	prefix := key[:prefixEnd]
	suffix := key[len(key)-4:]

	return prefix + "***" + suffix
}

// SanitizeIdentifier masks a user-provided identifier that may contain PII.
// If it looks like an email, it masks the email. Otherwise, it truncates.
// Example: "john@example.com" -> "j***@e******.c**"
// Example: "user_abc123xyz789" -> "user_abc...z789"
func SanitizeIdentifier(id string) string {
	if id == "" {
		return ""
	}

	// Check if it looks like an email
	if strings.Contains(id, "@") && strings.Contains(id, ".") {
		return MaskEmail(id)
	}

	// For non-email identifiers, truncate if long
	if len(id) > 16 {
		return TruncateID(id, 16)
	}

	return id
}

// TruncateID shortens an ID for readability while preserving enough for identification.
// Example: TruncateID("req_abc123xyz789def", 8) -> "req_abc1...9def"
//
// If the ID is shorter than maxLen, returns it unchanged.
func TruncateID(id string, maxLen int) string {
	if len(id) <= maxLen || maxLen < 8 {
		return id
	}

	// Keep first half and last quarter
	keepStart := maxLen / 2
	keepEnd := maxLen / 4

	return id[:keepStart] + "..." + id[len(id)-keepEnd:]
}

// maskString masks a string, keeping first char and adding asterisks.
func maskString(s string) string {
	if len(s) == 0 {
		return ""
	}
	if len(s) == 1 {
		return s
	}

	// Keep first character, mask the rest
	runes := []rune(s)
	var b strings.Builder
	b.Grow(len(runes))
	b.WriteRune(runes[0])
	for i := 1; i < len(runes); i++ {
		b.WriteByte('*')
	}
	return b.String()
}

// maskDomain masks a domain, keeping first char of each part.
func maskDomain(domain string) string {
	parts := strings.Split(domain, ".")
	masked := make([]string, len(parts))

	for i, part := range parts {
		if len(part) > 0 {
			runes := []rune(part)
			var b strings.Builder
			b.Grow(len(runes))
			b.WriteRune(runes[0])
			for j := 1; j < len(runes); j++ {
				b.WriteByte('*')
			}
			masked[i] = b.String()
		}
	}

	return strings.Join(masked, ".")
}
