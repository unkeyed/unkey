package deploy

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShellSingleQuote(t *testing.T) {
	t.Run("wraps plain value", func(t *testing.T) {
		require.Equal(t, "'value'", shellSingleQuote("value"))
	})

	t.Run("escapes embedded single quote", func(t *testing.T) {
		// it's -> 'it'\''s'
		require.Equal(t, `'it'\''s'`, shellSingleQuote("it's"))
	})

	t.Run("leaves newlines untouched inside quotes", func(t *testing.T) {
		require.Equal(t, "'a\nb'", shellSingleQuote("a\nb"))
	})
}

func TestBuildEnvFileSecret(t *testing.T) {
	t.Run("returns nil for no vars", func(t *testing.T) {
		require.Nil(t, buildEnvFileSecret(nil))
		require.Nil(t, buildEnvFileSecret(map[string]string{}))
	})

	t.Run("emits sorted single-quoted lines", func(t *testing.T) {
		out := buildEnvFileSecret(map[string]string{
			"B_KEY": "two",
			"A_KEY": "one",
		})
		require.Equal(t, "A_KEY='one'\nB_KEY='two'\n", string(out))
	})

	t.Run("is deterministic across calls", func(t *testing.T) {
		in := map[string]string{"FOO": "1", "BAR": "2", "BAZ": "3"}
		require.Equal(t, buildEnvFileSecret(in), buildEnvFileSecret(in))
	})
}

// TestBuildEnvFileSecret_RoundTripsThroughShell verifies the actual contract
// documented for users: the .env file is loaded with
// `set -a && . /run/secrets/.env && set +a`. We write the serialized bytes to a
// file, source it in a real shell, and assert every value comes back
// byte-for-byte, including multi-line values that the old .env format rejected.
func TestBuildEnvFileSecret_RoundTripsThroughShell(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh not available")
	}

	cases := map[string]string{
		"SIMPLE":       "value",
		"WITH_SPACES":  "value with spaces",
		"WITH_DOLLAR":  "ab$cd${HOME}",
		"WITH_DQUOTE":  `he said "hi"`,
		"WITH_BACKTCK": "echo `whoami`",
		"WITH_BSLASH":  `a\b\nc`,
		"WITH_HASH":    "value # not a comment",
		"WITH_SQUOTE":  "it's a 'value'",
		"WITH_EQUALS":  "key=val=more",
		"EMPTY":        "",
		"MULTILINE":    "line1\nline2\nline3",
		"PEM": "-----BEGIN PRIVATE KEY-----\n" +
			"MIIBVgIBADANBgkqhkiG9w0BAQEFAASCAUAwg==\n" +
			"-----END PRIVATE KEY-----",
		"JSON": "{\n  \"type\": \"service_account\",\n  \"id\": \"abc\"\n}",
		"CRLF": "carriage\r\nreturn",
	}

	envFile := buildEnvFileSecret(cases)
	require.NotNil(t, envFile)

	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	require.NoError(t, os.WriteFile(path, envFile, 0o600))

	for key, want := range cases {
		t.Run(key, func(t *testing.T) {
			// Source the file the same way the documented Dockerfile snippet does,
			// then print the variable with no added formatting.
			script := "set -a && . " + path + " && set +a && printf '%s' \"$" + key + "\""
			out, err := exec.Command("sh", "-c", script).Output()
			require.NoError(t, err)
			require.Equal(t, want, string(out))
		})
	}
}

func TestHashEnvVars(t *testing.T) {
	t.Run("empty for no vars", func(t *testing.T) {
		require.Empty(t, hashEnvVars(nil))
	})

	t.Run("stable regardless of map order", func(t *testing.T) {
		a := hashEnvVars(map[string]string{"A": "1", "B": "2"})
		b := hashEnvVars(map[string]string{"B": "2", "A": "1"})
		require.Equal(t, a, b)
	})

	t.Run("changes when a value changes", func(t *testing.T) {
		a := hashEnvVars(map[string]string{"A": "1"})
		b := hashEnvVars(map[string]string{"A": "2"})
		require.NotEqual(t, a, b)
	})
}
