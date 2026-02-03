package clickhouseuser

import (
	"fmt"
	"strings"
	"testing"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/require"
)

func TestGenerateSecurePassword_DSNCompatible(t *testing.T) {
	// Generate many passwords to increase chance of catching edge cases
	for range 100 {
		password, err := generateSecurePassword(passwordLength)
		require.NoError(t, err)

		// Build a DSN with this password and verify it parses correctly
		dsn := fmt.Sprintf("clickhouse://testuser:%s@localhost:9000/default", password)
		opts, err := ch.ParseDSN(dsn)
		require.NoError(t, err, "password broke DSN parsing: %q", password)

		// Verify the password round-trips correctly
		require.Equal(t, password, opts.Auth.Password, "password was mangled by DSN parsing")
	}
}

// TestGenerateSecurePassword_MeetsClickHouseCloudRequirements verifies passwords meet:
// - At least 12 characters long
// - At least 1 numeric character
// - At least 1 uppercase character
// - At least 1 lowercase character
// - At least 1 special character
func TestGenerateSecurePassword_MeetsClickHouseCloudRequirements(t *testing.T) {
	for range 100 {
		password, err := generateSecurePassword(passwordLength)
		require.NoError(t, err)

		require.GreaterOrEqual(t, len(password), 12, "password must be at least 12 characters")
		require.True(t, strings.ContainsAny(password, upper), "password must contain uppercase: %q", password)
		require.True(t, strings.ContainsAny(password, lower), "password must contain lowercase: %q", password)
		require.True(t, strings.ContainsAny(password, digits), "password must contain digit: %q", password)
		require.True(t, strings.ContainsAny(password, special), "password must contain special character: %q", password)
	}
}

func TestGenerateSecurePassword_MinLength(t *testing.T) {
	// Even if we request less than 12, it should enforce minimum
	password, err := generateSecurePassword(5)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(password), 12)
}
