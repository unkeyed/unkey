package keystore

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetRejectsEmptyNames(t *testing.T) {
	_, err := Get("", Account)
	require.Error(t, err)
	err = Set(Service, "", "secret")
	require.Error(t, err)
	err = Set(Service, Account, "")
	require.EqualError(t, err, "keystore: secret cannot be empty")
}

func TestSecretToolRoundTrip(t *testing.T) {
	bin := writeTool(t, secretToolScript)
	require.NoError(t, secretToolSet(bin, Service, Account, "unkey_from_store"))
	got, err := secretToolGet(bin, Service, Account)
	require.NoError(t, err)
	require.Equal(t, "unkey_from_store", got)
	require.NoError(t, secretToolDelete(bin, Service, Account))
	require.NoError(t, secretToolDelete(bin, Service, Account))
	_, err = secretToolGet(bin, Service, Account)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestLinuxStoreUsesSecretTool(t *testing.T) {
	bin := writeTool(t, secretToolScript)
	require.NoError(t, secretToolSet(bin, Service, Account, "unkey_platform"))
	got, err := secretToolGet(bin, Service, Account)
	require.NoError(t, err)
	require.Equal(t, "unkey_platform", got)
	require.NoError(t, secretToolDelete(bin, Service, Account))
	_, err = secretToolGet(bin, Service, Account)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestSecretToolMissingBinaryIsUnavailable(t *testing.T) {
	_, err := secretToolGet("unkey-secret-tool-missing", Service, Account)
	require.ErrorIs(t, err, ErrUnavailable)
	err = secretToolSet("unkey-secret-tool-missing", Service, Account, "unkey_x")
	require.ErrorIs(t, err, ErrUnavailable)
}

func TestSecretToolDBusFailureIsUnavailable(t *testing.T) {
	bin := writeTool(t, "#!/bin/sh\necho \"secret-tool: The name org.freedesktop.secrets was not provided by any .service files\" >&2\nexit 1\n")
	_, err := secretToolGet(bin, Service, Account)
	require.ErrorIs(t, err, ErrUnavailable)
}

func TestSecurityRoundTrip(t *testing.T) {
	bin := writeTool(t, securityScript)
	require.NoError(t, securitySet(bin, Service, Account, "unkey_from_keychain"))
	got, err := securityGet(bin, Service, Account)
	require.NoError(t, err)
	require.Equal(t, "unkey_from_keychain", got)
	require.NoError(t, securityDelete(bin, Service, Account))
	require.NoError(t, securityDelete(bin, Service, Account))
	_, err = securityGet(bin, Service, Account)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestSecuritySetKeepsTheSecretOutOfArgv(t *testing.T) {
	bin := writeTool(t, securityScript)
	secret := `unkey_with "quotes" and \backslash`
	require.NoError(t, securitySet(bin, Service, Account, secret))
	got, err := securityGet(bin, Service, Account)
	require.NoError(t, err)
	require.Equal(t, secret, got)

	argv, err := os.ReadFile(filepath.Join(os.Getenv("KEYSTORE_DIR"), "argv"))
	require.NoError(t, err)
	require.NotContains(t, string(argv), "unkey_with")
}

func TestSecuritySetDetectsASilentFailure(t *testing.T) {
	bin := writeTool(t, "#!/bin/sh\nif [ \"$1\" = \"-i\" ]; then cat >/dev/null; exit 0; fi\necho \"The specified item could not be found in the keychain.\" >&2\nexit 44\n")
	err := securitySet(bin, Service, Account, "unkey_x")
	require.ErrorIs(t, err, ErrUnavailable)
}

func TestSecuritySetRejectsLineBreaks(t *testing.T) {
	bin := writeTool(t, securityScript)
	require.Error(t, securitySet(bin, Service, Account, "unkey_x\nadd-generic-password"))
}

func TestSecurityInteractionIsUnavailable(t *testing.T) {
	bin := writeTool(t, "#!/bin/sh\necho \"security: User interaction is not allowed.\" >&2\nexit 36\n")
	err := securitySet(bin, Service, Account, "unkey_x")
	require.ErrorIs(t, err, ErrUnavailable)
}

func TestSecurityMissingBinaryIsUnavailable(t *testing.T) {
	_, err := securityGet("unkey-security-missing", Service, Account)
	require.ErrorIs(t, err, ErrUnavailable)
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
  IFS= read -r line
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
file="$dir/secret"
case "$cmd" in
  add-generic-password)
    printf '%s' "$password" > "$file"
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
