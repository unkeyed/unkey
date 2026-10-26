#!/usr/bin/env bash
set -euo pipefail

registry_script="$(git rev-parse --show-toplevel)/dev/depot-registry.sh"
scratch=$(mktemp -d)
trap 'rm -rf "$scratch"' EXIT
mkdir "$scratch/bin"
export PATH="$scratch/bin:$PATH"
export DEPOT_TEST_STATE="$scratch"
export GIT_CONFIG_NOSYSTEM=1 GIT_CONFIG_GLOBAL=/dev/null

cat > "$scratch/bin/depot" <<'BASH'
#!/usr/bin/env bash
set -euo pipefail
organization=${DEPOT_TOKEN%%:*}
state="$DEPOT_TEST_STATE/$organization.json"
if [ ! -f "$state" ]; then
  jq -n '[]' > "$state"
fi
case "$*" in
  'projects list --output json')
    if [ "${DEPOT_TEST_FAILURE:-}" = list ]; then
      echo 'permission_denied: Invalid token' >&2
      exit 1
    fi
    cat "$state"
    ;;
  *)
    if [ "$1" != projects ] || [ "$2" != create ]; then
      exit 1
    fi
    shift 2
    if [ "$1" != --organization ] || [ "$2" != '' ]; then
      echo 'The CLI default organization differs from the token organization' >&2
      exit 1
    fi
    shift 2
    if [ "$1" != --region ] || [ "$2" != us-east-1 ]; then
      exit 1
    fi
    shift 2
    if [ "${DEPOT_TEST_FAILURE:-}" = create ]; then
      echo 'permission_denied: Cannot create project' >&2
      exit 1
    fi
    id=$(head -c 8 /dev/urandom | od -A n -t x1 | tr -d ' \n')
    jq --arg id "$id" --arg name "$1" '. + [{id: $id, name: $name}]' "$state" > "$state.next"
    mv "$state.next" "$state"
    jq -n --arg id "$id" '{project_id: $id}'
    ;;
esac
BASH
chmod +x "$scratch/bin/depot"

git -C "$scratch" init --quiet
git -C "$scratch" config user.email 'developer-one@example.com'
cd "$scratch"
printf '%s\n' 'UNKEY_REGISTRY_PASSWORD=organizationa:first' > .env.depot

repository=$(bash "$registry_script")
project_id=$(jq -r '.[0].id' organizationa.json)
test "$repository" = "registry.depot.dev/$project_id"
test "$(jq length organizationa.json)" = 1
test "$(bash "$registry_script")" = "$repository"
test "$(jq length organizationa.json)" = 1
echo 'PASS: create once and reuse the returned registry'

printf '%s\n' 'UNKEY_REGISTRY_PASSWORD=organizationa:rotated' > .env.depot
test "$(bash "$registry_script")" = "$repository"
test "$(jq length organizationa.json)" = 1
echo 'PASS: token rotation in the same organization reuses the registry'

printf '%s\n' 'UNKEY_REGISTRY_PASSWORD=organizationa:$(touch should-not-exist)' > .env.depot
test "$(bash "$registry_script")" = "$repository"
test ! -e should-not-exist
echo 'PASS: environment-file values are parsed without executing shell code'

git config user.email 'developer-two@example.com'
repository_other=$(bash "$registry_script")
test "$repository_other" != "$repository"
test "$(jq length organizationa.json)" = 2
echo 'PASS: developers get separate registry projects'

printf '%s\n' 'UNKEY_REGISTRY_PASSWORD=organizationb:first' > .env.depot
repository_org=$(bash "$registry_script")
test "$repository_org" != "$repository_other"
test "$(jq length organizationb.json)" = 1
echo 'PASS: switching organizations selects a registry in the new organization'

expect_failure() {
  if bash "$registry_script" > "$scratch/output" 2> "$scratch/error"; then
    echo 'Expected registry setup to fail' >&2
    exit 1
  fi
  test ! -s "$scratch/output"
  grep -F "$1" "$scratch/error" > /dev/null
}

DEPOT_TEST_FAILURE=list expect_failure 'Cannot list Depot projects'
test "$(jq length organizationb.json)" = 1
echo 'PASS: list failure does not create a registry or return a fallback'

git config user.email 'developer-three@example.com'
DEPOT_TEST_FAILURE=create expect_failure 'Cannot create project'
test "$(jq length organizationb.json)" = 1
echo 'PASS: project creation failure stops setup'

git config user.email 'developer-two@example.com'
jq '. + .' organizationb.json > duplicate.json
mv duplicate.json organizationb.json
expect_failure 'Multiple Depot projects match this developer'
echo 'PASS: duplicate project names are rejected'

jq '.[0:1] | .[0].id = "invalid/path"' organizationb.json > invalid.json
mv invalid.json organizationb.json
expect_failure 'Depot returned an invalid project ID'
echo 'PASS: malformed registry IDs are rejected'

git config --unset user.email
expect_failure 'Set git config user.email'
echo 'PASS: missing developer identity fails before provisioning'

git config user.email 'developer-one@example.com'
: > .env.depot
UNKEY_REGISTRY_PASSWORD=organizationa:inherited expect_failure 'Set UNKEY_REGISTRY_PASSWORD'
echo 'PASS: missing file credential does not use an inherited token'
