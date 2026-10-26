#!/usr/bin/env bash
set -euo pipefail

credentials=$(kubectl create secret generic depot-credentials --from-env-file=./.env.depot --dry-run=client -o json)
DEPOT_TOKEN=$(jq -r '.data.UNKEY_REGISTRY_PASSWORD // "" | @base64d' <<< "$credentials")
if [ -z "$DEPOT_TOKEN" ]; then
  echo "Set UNKEY_REGISTRY_PASSWORD in dev/.env.depot to a Depot organization API token." >&2
  exit 1
fi
export DEPOT_TOKEN

if ! identity=$(git config user.email) || [ -z "$identity" ]; then
  echo "Set git config user.email to identify your local Depot registry project." >&2
  exit 1
fi
if command -v sha256sum >/dev/null 2>&1; then
  checksum_cmd=(sha256sum)
else
  checksum_cmd=(shasum -a 256)
fi
identity_hash=$(printf '%s' "$identity" | "${checksum_cmd[@]}" | cut -c1-16)
project_name="unkey-local-registry-$identity_hash"

if ! projects=$(depot projects list --output json); then
  echo "Cannot list Depot projects. Check the organization API token in dev/.env.depot." >&2
  exit 1
fi
project_id=$(jq -er --arg name "$project_name" '
  [.[] | select(.name == $name)] |
  if length > 1 then error("Multiple Depot projects match this developer; remove the duplicate before continuing")
  elif length == 1 then .[0].id
  else "" end
' <<< "$projects")

if [ -z "$project_id" ]; then
  project=$(depot projects create --organization '' --region us-east-1 "$project_name")
  project_id=$(jq -er '.project_id' <<< "$project")
fi

if [[ ! "$project_id" =~ ^[a-z0-9]+$ ]]; then
  echo "Depot returned an invalid project ID." >&2
  exit 1
fi

printf 'registry.depot.dev/%s\n' "$project_id"
