package deploy

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// buildEnvFileSecret serializes env vars into a .env-formatted byte slice for
// injection as a BuildKit secret, mounted at /run/secrets/.env. Returns nil
// when there are no env vars.
//
// Values are POSIX-quoted per quoteEnvValue so the file round-trips exactly
// through shell sourcing (set -a && . /run/secrets/.env && set +a) and through
// mainstream dotenv parsers. This is what lets values carry newlines,
// whitespace, and shell metacharacters that a bare KEY=value line could not
// represent. Keys are emitted in sorted order so identical inputs produce
// identical bytes, which the env hash depends on.
func buildEnvFileSecret(envVars map[string]string) []byte {
	if len(envVars) == 0 {
		return nil
	}

	keys := make([]string, 0, len(envVars))
	for k := range envVars {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var buf strings.Builder
	for _, k := range keys {
		buf.WriteString(k)
		buf.WriteByte('=')
		buf.WriteString(quoteEnvValue(envVars[k]))
		buf.WriteByte('\n')
	}
	return []byte(buf.String())
}

// envValueBareAllowed is the shlex-style safe set: bytes that carry no meaning
// to the shell or to a dotenv parser, so a value composed only of these can be
// emitted unquoted and read back byte-identical. Any byte >= 0x80 (multibyte
// UTF-8) falls outside it, so unicode values take a quoted path.
func envValueBareAllowed(b byte) bool {
	switch {
	case b >= 'A' && b <= 'Z':
		return true
	case b >= 'a' && b <= 'z':
		return true
	case b >= '0' && b <= '9':
		return true
	}
	switch b {
	case '_', '@', '%', '+', '=', ':', ',', '.', '/', '-':
		return true
	default:
		return false
	}
}

// isBareEnvValue reports whether v can be emitted unquoted. The empty string
// qualifies so KEY= sets an empty value.
func isBareEnvValue(v string) bool {
	for i := 0; i < len(v); i++ {
		if !envValueBareAllowed(v[i]) {
			return false
		}
	}
	return true
}

// quoteEnvValue renders a single .env value with the least quoting that
// survives POSIX sourcing and mainstream dotenv parsers intact:
//
//   - bare when every byte is in the safe set (existing simple values stay
//     byte-identical);
//   - single-quoted when the value carries metacharacters but no single quote,
//     so every byte inside is literal (newlines, $, ", \, # included); this is
//     the exact form for PEM blocks and for JSON, which rarely contains ';
//   - double-quoted when the value contains a single quote, backslash-escaping
//     the four bytes the shell still interprets inside double quotes. Literal
//     newlines stay literal. This is exact in the shell; it diverges from some
//     dotenv parsers only for values that mix ' with other metacharacters.
//
// The close-quote backslash-quote reopen trick that shells accept is
// deliberately avoided: it is shell-correct but unreadable by every mainstream
// dotenv parser.
func quoteEnvValue(v string) string {
	if isBareEnvValue(v) {
		return v
	}
	if !strings.Contains(v, "'") {
		return "'" + v + "'"
	}

	var b strings.Builder
	b.Grow(len(v) + 2)
	b.WriteByte('"')
	for i := 0; i < len(v); i++ {
		c := v[i]
		if c == '\\' || c == '"' || c == '$' || c == '`' {
			b.WriteByte('\\')
		}
		b.WriteByte(c)
	}
	b.WriteByte('"')
	return b.String()
}

// hashEnvVars returns a hex-encoded SHA-256 hash of the env vars, stable across
// runs and safe to embed in image metadata without exposing values. Returns an
// empty string when there are no env vars.
//
// Each key and value is length-prefixed before hashing so no combination of
// key and value bytes can encode to the same stream as a different map. A
// delimiter-joined encoding lets a value that contains the delimiter collide
// with a different set of pairs, and a collision makes BuildKit silently reuse
// a RUN layer built with stale secrets, the exact failure this hash prevents.
func hashEnvVars(envVars map[string]string) string {
	if len(envVars) == 0 {
		return ""
	}

	keys := make([]string, 0, len(envVars))
	for k := range envVars {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	h := sha256.New()
	for _, k := range keys {
		v := envVars[k]
		fmt.Fprintf(h, "%d:%s=%d:%s;", len(k), k, len(v), v)
	}
	return hex.EncodeToString(h.Sum(nil))
}
