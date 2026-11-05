package tools

import (
	"regexp"
	"strings"
	"unicode"
)

// namePrefix is shared by every toolset because Speakeasy published compute
// tools as unkey_v2_* too, and users have these names in allowlists.
const namePrefix = "unkey_v2_"

var nameRule = regexp.MustCompile(`^[a-z0-9_]{1,64}$`)

// toolName reproduces Speakeasy's rule, namePrefix + snake_case(operationId),
// so "deployments.createDeploymentV3" becomes
// "unkey_v2_deployments_create_deployment_v3". A digit never starts a word.
func toolName(operationID string) string {
	var b strings.Builder
	b.WriteString(namePrefix)
	runes := []rune(operationID)
	for i, r := range runes {
		if r == '.' || r == '-' {
			b.WriteRune('_')
			continue
		}
		if !unicode.IsUpper(r) {
			b.WriteRune(r)
			continue
		}
		if i > 0 && runes[i-1] != '.' && (unicode.IsLower(runes[i-1]) || unicode.IsDigit(runes[i-1]) || (i+1 < len(runes) && unicode.IsLower(runes[i+1]))) {
			b.WriteRune('_')
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}
