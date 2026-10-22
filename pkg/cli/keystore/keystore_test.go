package keystore

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
)

func TestGetRejectsEmptyNames(t *testing.T) {
	_, err := Get("", rootKeyAccount)
	require.Error(t, err)
	err = Set(serviceName, "", "secret")
	require.Error(t, err)
	err = Set(serviceName, rootKeyAccount, "")
	require.EqualError(t, err, "keystore: secret cannot be empty")
}

func TestSecretToolRoundTrip(t *testing.T) {
	bin := writeTool(t, secretToolScript)
	require.NoError(t, secretToolSet(bin, serviceName, rootKeyAccount, "unkey_from_store"))
	got, err := secretToolGet(bin, serviceName, rootKeyAccount)
	require.NoError(t, err)
	require.Equal(t, "unkey_from_store", got)
	require.NoError(t, secretToolDelete(bin, serviceName, rootKeyAccount))
	require.NoError(t, secretToolDelete(bin, serviceName, rootKeyAccount))
	_, err = secretToolGet(bin, serviceName, rootKeyAccount)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestLinuxStoreUsesSecretTool(t *testing.T) {
	bin := writeTool(t, secretToolScript)
	require.NoError(t, secretToolSet(bin, serviceName, rootKeyAccount, "unkey_platform"))
	got, err := secretToolGet(bin, serviceName, rootKeyAccount)
	require.NoError(t, err)
	require.Equal(t, "unkey_platform", got)
	require.NoError(t, secretToolDelete(bin, serviceName, rootKeyAccount))
	_, err = secretToolGet(bin, serviceName, rootKeyAccount)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestSecretToolMissingBinaryIsUnavailable(t *testing.T) {
	_, err := secretToolGet("unkey-secret-tool-missing", serviceName, rootKeyAccount)
	require.ErrorIs(t, err, ErrUnavailable)
	err = secretToolSet("unkey-secret-tool-missing", serviceName, rootKeyAccount, "unkey_x")
	require.ErrorIs(t, err, ErrUnavailable)
}

func TestSecretToolWriteFailureIsNotAMissingItem(t *testing.T) {
	bin := writeTool(t, "#!/bin/sh\nexit 1\n")
	err := secretToolSet(bin, serviceName, rootKeyAccount, "unkey_x")
	require.Error(t, err)
	require.NotErrorIs(t, err, ErrNotFound)
	require.NotErrorIs(t, err, ErrUnavailable)
}

func TestWindowsCredentialLimitMatchesMeasuredSessions(t *testing.T) {
	require.False(t, ExceedsWindowsCredentialBlob(string(bytes.Repeat([]byte("a"), 761))))
	require.True(t, ExceedsWindowsCredentialBlob(string(bytes.Repeat([]byte("a"), 2850))))
	require.True(t, ExceedsWindowsCredentialBlob(string(bytes.Repeat([]byte("a"), 6111))))
}

func TestSecretToolDBusFailureIsUnavailable(t *testing.T) {
	bin := writeTool(t, "#!/bin/sh\necho \"secret-tool: The name org.freedesktop.secrets was not provided by any .service files\" >&2\nexit 1\n")
	_, err := secretToolGet(bin, serviceName, rootKeyAccount)
	require.ErrorIs(t, err, ErrUnavailable)
}

func TestSecurityRoundTrip(t *testing.T) {
	bin := writeTool(t, securityScript)
	require.NoError(t, securitySet(bin, serviceName, rootKeyAccount, "unkey_from_keychain"))
	got, err := securityGet(bin, serviceName, rootKeyAccount)
	require.NoError(t, err)
	require.Equal(t, "unkey_from_keychain", got)
	require.NoError(t, securityDelete(bin, serviceName, rootKeyAccount))
	require.NoError(t, securityDelete(bin, serviceName, rootKeyAccount))
	_, err = securityGet(bin, serviceName, rootKeyAccount)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestSecuritySetKeepsTheSecretOutOfArgv(t *testing.T) {
	bin := writeTool(t, securityScript)
	secret := `unkey_with "quotes" and \backslash`
	require.NoError(t, securitySet(bin, serviceName, rootKeyAccount, secret))
	got, err := securityGet(bin, serviceName, rootKeyAccount)
	require.NoError(t, err)
	require.Equal(t, secret, got)

	argv, err := os.ReadFile(filepath.Join(os.Getenv("KEYSTORE_DIR"), "argv"))
	require.NoError(t, err)
	require.NotContains(t, string(argv), "unkey_with")
}

func TestSecuritySetDetectsASilentFailure(t *testing.T) {
	bin := writeTool(t, "#!/bin/sh\nif [ \"$1\" = \"-i\" ]; then cat >/dev/null; exit 0; fi\necho \"The specified item could not be found in the keychain.\" >&2\nexit 44\n")
	err := securitySet(bin, serviceName, rootKeyAccount, "unkey_x")
	require.ErrorIs(t, err, ErrUnavailable)
}

func TestSecuritySetRejectsLineBreaks(t *testing.T) {
	bin := writeTool(t, securityScript)
	require.Error(t, securitySet(bin, serviceName, rootKeyAccount, "unkey_x\nadd-generic-password"))
}

// TestSecuritySetRoundTripsABlobLargerThanTheInteractiveLine guarantees a CLI
// session bigger than macOS `security -i` can accept on one line is stored
// whole, off argv, and comes back unchanged.
func TestSecuritySetRoundTripsABlobLargerThanTheInteractiveLine(t *testing.T) {
	bin := writeTool(t, securityScript)
	secret := `{"active":"ws_1","sessions":{"ws_1":{"access_token":"` + strings.Repeat("a", 5000) + `","refresh_token":"ref\"resh\\token"}}}`
	require.Greater(t, len(secret), securityMaxLineLen)

	require.NoError(t, securitySet(bin, serviceName, sessionAccount, secret))
	got, err := securityGet(bin, serviceName, sessionAccount)
	require.NoError(t, err)
	require.Equal(t, secret, got)

	dir := os.Getenv("KEYSTORE_DIR")
	argv, err := os.ReadFile(filepath.Join(dir, "argv"))
	require.NoError(t, err)
	require.NotContains(t, string(argv), "access_token")

	lines, err := os.ReadFile(filepath.Join(dir, "lines"))
	require.NoError(t, err)
	for _, line := range strings.Split(strings.TrimSuffix(string(lines), "\n"), "\n") {
		require.LessOrEqual(t, len(line), securityMaxLineLen)
		require.NotEmpty(t, line)
	}

	require.NoError(t, securityDelete(bin, serviceName, sessionAccount))
	_, err = securityGet(bin, serviceName, sessionAccount)
	require.ErrorIs(t, err, ErrNotFound)
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	for _, entry := range entries {
		require.NotContains(t, entry.Name(), sessionAccount)
	}
}

// TestSecuritySetDeletesATruncatedBlob guarantees a tool that silently shortens
// the secret does not leave that value for the next read.
func TestSecuritySetDeletesATruncatedBlob(t *testing.T) {
	bin := writeTool(t, truncatingSecurityScript)
	secret := strings.Repeat("x", 80)
	err := securitySet(bin, serviceName, sessionAccount, secret)
	require.ErrorIs(t, err, ErrUnavailable)
	_, err = securityGet(bin, serviceName, sessionAccount)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestSecurityGetMissingChunkIsNotFound(t *testing.T) {
	bin := writeTool(t, securityScript)
	plantItem(t, sessionAccount, chunkHeader+"2")
	plantItem(t, chunkAccount(sessionAccount, 0), "kept")

	_, err := securityGet(bin, serviceName, sessionAccount)
	require.ErrorIs(t, err, ErrNotFound)
	require.ErrorContains(t, err, "unreadable")
	require.NoFileExists(t, itemPath(t, sessionAccount))
	require.NoFileExists(t, itemPath(t, chunkAccount(sessionAccount, 0)))
}

func TestSecuritySetChunkWriteFailureKeepsThePreviousSecret(t *testing.T) {
	bin := writeTool(t, securityScript)
	plantItem(t, rootKeyAccount, "unkey_previous")
	t.Setenv("KEYSTORE_FAIL_ACCOUNT", chunkAccount(rootKeyAccount, 1))

	err := securitySet(bin, serviceName, rootKeyAccount, strings.Repeat("a", 5000))
	require.Error(t, err)
	require.NotErrorIs(t, err, ErrNotFound)

	body, readErr := os.ReadFile(itemPath(t, rootKeyAccount))
	require.NoError(t, readErr)
	require.Equal(t, "unkey_previous", string(body))
	require.NoFileExists(t, itemPath(t, chunkAccount(rootKeyAccount, 0)))
}

func TestSecurityGetReadErrorKeepsStoredItems(t *testing.T) {
	bin := writeTool(t, securityScript)
	plantItem(t, sessionAccount, chunkHeader+"1")
	plantItem(t, chunkAccount(sessionAccount, 0), "piece")
	t.Setenv("KEYSTORE_FAIL_READ_CHUNK", "1")

	_, err := securityGet(bin, serviceName, sessionAccount)
	require.Error(t, err)
	require.NotErrorIs(t, err, ErrNotFound)

	body, readErr := os.ReadFile(itemPath(t, sessionAccount))
	require.NoError(t, readErr)
	require.Equal(t, chunkHeader+"1", string(body))
	piece, readErr := os.ReadFile(itemPath(t, chunkAccount(sessionAccount, 0)))
	require.NoError(t, readErr)
	require.Equal(t, "piece", string(piece))
}

func TestSecuritySetReadBackErrorKeepsTheSecret(t *testing.T) {
	bin := writeTool(t, securityScript)
	t.Setenv("KEYSTORE_FAIL_READ", "1")
	secret := "unkey_written"

	err := securitySet(bin, serviceName, rootKeyAccount, secret)
	require.Error(t, err)
	require.NotErrorIs(t, err, ErrNotFound)

	body, readErr := os.ReadFile(itemPath(t, rootKeyAccount))
	require.NoError(t, readErr)
	require.Equal(t, secret, string(body))
}

func TestSecurityDeleteRemovesLeftoverChunks(t *testing.T) {
	bin := writeTool(t, securityScript)
	plantItem(t, sessionAccount, chunkHeader+"1")
	plantItem(t, chunkAccount(sessionAccount, 0), "zero")
	plantItem(t, chunkAccount(sessionAccount, 1), "one")
	plantItem(t, chunkAccount(sessionAccount, 2), "two")
	plantItem(t, rootKeyAccount, "unkey_keep")

	require.NoError(t, securityDelete(bin, serviceName, sessionAccount))
	require.NoFileExists(t, itemPath(t, sessionAccount))
	require.NoFileExists(t, itemPath(t, chunkAccount(sessionAccount, 0)))
	require.NoFileExists(t, itemPath(t, chunkAccount(sessionAccount, 1)))
	require.NoFileExists(t, itemPath(t, chunkAccount(sessionAccount, 2)))
	body, err := os.ReadFile(itemPath(t, rootKeyAccount))
	require.NoError(t, err)
	require.Equal(t, "unkey_keep", string(body))
}

func TestSecurityGetRejectsAnUnparseableChunkHeader(t *testing.T) {
	bin := writeTool(t, securityScript)
	plantItem(t, rootKeyAccount, chunkHeader+"nope")

	got, err := securityGet(bin, serviceName, rootKeyAccount)
	require.EqualError(t, err, "security: unparseable keychain chunk header")
	require.Empty(t, got)
	body, readErr := os.ReadFile(itemPath(t, rootKeyAccount))
	require.NoError(t, readErr)
	require.Equal(t, chunkHeader+"nope", string(body))
}

func TestPlanChunksStayUnderThePieceSizeAndCutOnUTF8(t *testing.T) {
	secret := strings.Repeat("a", 8000)
	parts, chunked, err := planChunks(serviceName, sessionAccount, secret)
	require.NoError(t, err)
	require.True(t, chunked)
	require.Equal(t, secret, strings.Join(parts, ""))
	require.Equal(t, chunkPieceBytes, len(parts[0]))
	for _, part := range parts {
		require.LessOrEqual(t, len(part), chunkPieceBytes)
		require.True(t, utf8.ValidString(part))
	}

	euro := "€"
	split := strings.Repeat("a", chunkPieceBytes-1) + euro + strings.Repeat("b", 2000)
	parts, chunked, err = planChunks(serviceName, sessionAccount, split)
	require.NoError(t, err)
	require.True(t, chunked)
	require.Equal(t, split, strings.Join(parts, ""))
	require.Equal(t, strings.Repeat("a", chunkPieceBytes-1), parts[0])
	require.True(t, strings.HasPrefix(parts[1], euro))
	for _, part := range parts {
		require.LessOrEqual(t, len(part), chunkPieceBytes)
		require.True(t, utf8.ValidString(part))
	}
}

func TestSecurityInteractionIsUnavailable(t *testing.T) {
	bin := writeTool(t, "#!/bin/sh\necho \"security: User interaction is not allowed.\" >&2\nexit 36\n")
	err := securitySet(bin, serviceName, rootKeyAccount, "unkey_x")
	require.ErrorIs(t, err, ErrUnavailable)
}

func TestSecurityMissingBinaryIsUnavailable(t *testing.T) {
	_, err := securityGet("unkey-security-missing", serviceName, rootKeyAccount)
	require.ErrorIs(t, err, ErrUnavailable)
}

func plantItem(t *testing.T, account, secret string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(os.Getenv("KEYSTORE_DIR"), 0o700))
	require.NoError(t, os.WriteFile(itemPath(t, account), []byte(secret), 0o600))
}

func itemPath(t *testing.T, account string) string {
	t.Helper()
	return filepath.Join(os.Getenv("KEYSTORE_DIR"), strings.ReplaceAll(account, "/", "_"))
}

func writeTool(t *testing.T, script string) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("KEYSTORE_DIR", filepath.Join(dir, "items"))
	path := filepath.Join(dir, "tool")
	require.NoError(t, os.WriteFile(path, []byte(script), 0o755))
	return path
}

const secretToolScript = `#!/bin/sh
set -eu
dir="${KEYSTORE_DIR}"
mkdir -p "$dir"
cmd="$1"
shift
case "$cmd" in
  lookup|clear)
    if [ "$1" != "service" ] || [ "$3" != "account" ]; then
      echo "bad attributes: $*" >&2
      exit 2
    fi
    ;;
  store)
    if [ "$1" != "--label=Unkey root key" ] || [ "$2" != "service" ] || [ "$4" != "account" ]; then
      echo "bad store args: $*" >&2
      exit 2
    fi
    ;;
  *)
    echo "unknown $cmd" >&2
    exit 2
    ;;
esac
case "$cmd" in
  lookup)
    if [ ! -f "$dir/secret" ]; then
      exit 1
    fi
    cat "$dir/secret"
    ;;
  store)
    cat > "$dir/secret"
    ;;
  clear)
    rm -f "$dir/secret"
    ;;
esac
`

const securityScript = `#!/bin/sh
set -eu
dir="${KEYSTORE_DIR}"
mkdir -p "$dir"
printf '%s\n' "$*" >> "$dir/argv"
if [ "$1" = "-i" ]; then
  IFS= read -r line || exit 2
  printf '%s\n' "$line" >> "$dir/lines"
  if [ ${#line} -gt 4095 ]; then
    echo "interactive line exceeds 4095 bytes" >&2
    exit 1
  fi
  eval "set -- $line"
fi
cmd="$1"
shift
service=""
account=""
password=""
while [ $# -gt 0 ]; do
  case "$1" in
    -U) shift ;;
    -s) service="$2"; shift 2 ;;
    -a) account="$2"; shift 2 ;;
    -w)
      if [ "$cmd" = "add-generic-password" ]; then
        password="$2"
        shift 2
      else
        shift
      fi
      ;;
    *)
      echo "bad arg $1" >&2
      exit 2
      ;;
  esac
done
if [ -z "$service" ] || [ -z "$account" ]; then
  echo "missing service or account" >&2
  exit 2
fi
file="$dir/$(printf '%s' "$account" | tr '/' '_')"
case "$cmd" in
  add-generic-password)
    if [ "${KEYSTORE_FAIL_ACCOUNT:-}" = "$account" ]; then
      echo "write failed" >&2
      exit 1
    fi
    printf '%s' "$password" > "$file"
    ;;
  find-generic-password)
    if [ "${KEYSTORE_FAIL_READ:-}" = "1" ]; then
      echo "temporary keychain failure" >&2
      exit 1
    fi
    case "$account" in
      */*)
        if [ "${KEYSTORE_FAIL_READ_CHUNK:-}" = "1" ]; then
          echo "temporary keychain failure" >&2
          exit 1
        fi
        ;;
    esac
    if [ ! -f "$file" ]; then
      echo "The specified item could not be found in the keychain." >&2
      exit 44
    fi
    cat "$file"
    printf '\n'
    ;;
  delete-generic-password)
    if [ ! -f "$file" ]; then
      echo "The specified item could not be found in the keychain." >&2
      exit 44
    fi
    rm -f "$file"
    ;;
  *)
    echo "unknown $cmd" >&2
    exit 2
    ;;
esac
`

// truncatingSecurityScript accepts a write and keeps only the first 10 bytes,
// the way `security -i` keeps a prefix of an overlong command.
const truncatingSecurityScript = `#!/bin/sh
set -eu
dir="${KEYSTORE_DIR}"
mkdir -p "$dir"
if [ "$1" = "-i" ]; then
  IFS= read -r line || exit 2
  eval "set -- $line"
fi
cmd="$1"
shift
service=""
account=""
password=""
while [ $# -gt 0 ]; do
  case "$1" in
    -U) shift ;;
    -s) service="$2"; shift 2 ;;
    -a) account="$2"; shift 2 ;;
    -w)
      if [ "$cmd" = "add-generic-password" ]; then
        password="$2"
        shift 2
      else
        shift
      fi
      ;;
    *)
      echo "bad arg $1" >&2
      exit 2
      ;;
  esac
done
file="$dir/$(printf '%s' "$account" | tr '/' '_')"
case "$cmd" in
  add-generic-password)
    printf '%s' "$password" | cut -c1-10 > "$file"
    ;;
  find-generic-password)
    if [ ! -f "$file" ]; then
      echo "The specified item could not be found in the keychain." >&2
      exit 44
    fi
    cat "$file"
    printf '\n'
    ;;
  delete-generic-password)
    if [ ! -f "$file" ]; then
      echo "The specified item could not be found in the keychain." >&2
      exit 44
    fi
    rm -f "$file"
    ;;
  *)
    echo "unknown $cmd" >&2
    exit 2
    ;;
esac
`
