package jwt

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestNormalizeAudience_AcceptsStringAndArray guarantees WorkOS Connect's
// string aud and this package's array aud both decode into Audience.
func TestNormalizeAudience_AcceptsStringAndArray(t *testing.T) {
	t.Parallel()

	stringAud, err := normalizeAudience([]byte(`{"aud":"https://mcp.unkey.com/mcp/v2","sub":"user"}`))
	require.NoError(t, err)
	var fromString RegisteredClaims
	require.NoError(t, json.Unmarshal(stringAud, &fromString))
	require.Equal(t, []string{"https://mcp.unkey.com/mcp/v2"}, fromString.Audience)
	require.Equal(t, "user", fromString.Subject)

	original := []byte(`{"aud":["https://mcp.unkey.com/mcp/v2","api.unkey.com"]}`)
	arrayAud, err := normalizeAudience(original)
	require.NoError(t, err)
	require.Equal(t, original, arrayAud)
}
