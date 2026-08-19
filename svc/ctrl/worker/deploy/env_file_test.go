package deploy

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildEnvFileSecret_Serialization(t *testing.T) {
	require.Nil(t, buildEnvFileSecret(nil))
	require.Nil(t, buildEnvFileSecret(map[string]string{}))

	tests := []struct {
		name string
		in   map[string]string
		want string
	}{
		{"bare simple value", map[string]string{"FOO": "bar"}, "FOO=bar\n"},
		{"bare allowlist chars stay unquoted", map[string]string{"URL": "https://a.b/c:1,d=e+f-g_@%.x"}, "URL=https://a.b/c:1,d=e+f-g_@%.x\n"},
		{"empty value stays bare", map[string]string{"EMPTY": ""}, "EMPTY=\n"},
		{"spaces force single quoting", map[string]string{"K": "a b"}, "K='a b'\n"},
		{"dollar forces single quoting", map[string]string{"K": "a$b"}, "K='a$b'\n"},
		{"hash forces single quoting", map[string]string{"K": "a#b"}, "K='a#b'\n"},
		{"newline forces single quoting", map[string]string{"K": "a\nb"}, "K='a\nb'\n"},
		{"double quote uses single quoting", map[string]string{"K": `a"b`}, "K='a\"b'\n"},
		{"single quote falls back to double quoting", map[string]string{"K": "a'b"}, "K=\"a'b\"\n"},
		{"keys are emitted in sorted order", map[string]string{"B": "2", "A": "1"}, "A=1\nB=2\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, string(buildEnvFileSecret(tt.in)))
		})
	}
}

// TestBuildEnvFileSecret_DoubleQuoteEscaping pins the exact escaping of the
// double-quote fallback: only backslash, double quote, dollar, and backtick
// gain a leading backslash; a literal single quote inside stays bare.
func TestBuildEnvFileSecret_DoubleQuoteEscaping(t *testing.T) {
	input := "a'\"$`\\b"
	want := "\"" + "a" + "'" + "\\\"" + "\\$" + "\\`" + "\\\\" + "b" + "\"" + "\n"
	require.Equal(t, "K="+want, string(buildEnvFileSecret(map[string]string{"K": input})))
}

// TestBuildEnvFileSecret_ShellRoundTrip is the load-bearing correctness check:
// values written to a .env file and sourced by a POSIX shell must come back
// byte-identical. Sourcing with the documented set -a pattern and dumping each
// variable NUL-delimited avoids any newline ambiguity in the comparison.
func TestBuildEnvFileSecret_ShellRoundTrip(t *testing.T) {
	envVars := map[string]string{
		"PEM":         "-----BEGIN PRIVATE KEY-----\nMIIabc/def+123==\nghi/JKL+456==\n-----END PRIVATE KEY-----\n",
		"JSONVAL":     `{"a":"b","nested":{"x":1,"y":[2,3]}}`,
		"QUOTEDOLLAR": "it's $HOME and `date` cost",
		"SPACES":      "hello world  two  spaces",
		"HASHVAL":     "value#notacomment",
		"BACKSLASH":   `a\b\c\n`,
		"CRLF":        "line1\r\nline2",
		"EMPTY":       "",
		"UNICODE":     "café ☕ 日本語",
	}

	out := buildEnvFileSecret(envVars)
	require.NotNil(t, out)

	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	require.NoError(t, os.WriteFile(path, out, 0o600))

	keys := make([]string, 0, len(envVars))
	for k := range envVars {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var refs strings.Builder
	for _, k := range keys {
		refs.WriteString(` "$`)
		refs.WriteString(k)
		refs.WriteString(`"`)
	}
	script := `set -a; . "$1"; set +a; printf '%s\0'` + refs.String()

	cmd := exec.Command("sh", "-c", script, "sh", path)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	require.NoErrorf(t, cmd.Run(), "sh failed: %s", stderr.String())

	// printf emits a trailing NUL after the final value, so the split yields one
	// empty trailing element.
	got := strings.Split(stdout.String(), "\x00")
	require.Len(t, got, len(keys)+1)
	require.Equal(t, "", got[len(keys)])
	for i, k := range keys {
		require.Equalf(t, envVars[k], got[i], "value for %s did not round-trip through the shell", k)
	}
}

// TestBuildEnvFileSecret_DotenvShape asserts every quoted entry matches the
// quoted alternative of node dotenv's LINE regex rather than its bare fallback,
// pinning dotenv compatibility without running Node in CI.
func TestBuildEnvFileSecret_DotenvShape(t *testing.T) {
	lineRegex := regexp.MustCompile(`(?m)^\s*(?:export\s+)?([\w.-]+)(?:\s*=\s*?|:\s+?)(\s*'(?:\\'|[^'])*'|\s*"(?:\\"|[^"])*"|\s*` + "`" + `(?:\\` + "`" + `|[^` + "`" + `])*` + "`" + `|[^#\r\n]+)?\s*(?:#.*)?$`)

	quotedValues := map[string]string{
		"PEM":     "-----BEGIN PRIVATE KEY-----\nMIIabc/def+123==\n-----END PRIVATE KEY-----\n",
		"JSONVAL": `{"a":"b","n":1}`,
		"SPACES":  "hello world",
		"HASHVAL": "value#notacomment",
		"SQUOTE":  "it's fine",
	}
	for k, v := range quotedValues {
		t.Run(k, func(t *testing.T) {
			entry := strings.TrimRight(string(buildEnvFileSecret(map[string]string{k: v})), "\n")
			m := lineRegex.FindStringSubmatch(entry)
			require.NotNilf(t, m, "entry did not match dotenv LINE regex: %q", entry)
			require.Equal(t, k, m[1], "key capture mismatch")

			valueGroup := m[2]
			require.NotEmpty(t, valueGroup, "value capture was empty")
			first := valueGroup[0]
			require.Truef(t, first == '\'' || first == '"',
				"value was matched by the bare fallback, not the quoted alternative: %q", valueGroup)
		})
	}
}

func TestHashEnvVars_Unambiguous(t *testing.T) {
	require.Empty(t, hashEnvVars(nil))
	require.Empty(t, hashEnvVars(map[string]string{}))

	// A delimiter-joined encoding would let a newline inside one value collide
	// with a separate key. Length-prefixing keeps them distinct.
	collidable := hashEnvVars(map[string]string{"A": "1\nB=2"})
	separate := hashEnvVars(map[string]string{"A": "1", "B": "2"})
	require.NotEqual(t, collidable, separate)

	// The hash is stable regardless of map iteration order.
	require.Equal(t, separate, hashEnvVars(map[string]string{"B": "2", "A": "1"}))
}
