#!/usr/bin/env bash
# Closes deploy requests and deletes PlanetScale branches whose GitHub PR is
# closed (merged or abandoned). The per-PR event cleanup only fires when a PR
# closes, so branches orphaned by an earlier merge or by a cleanup run that
# never finished keep their deploy requests armed forever. This scheduled sweep
# reconciles pr-* branches against GitHub PR state and tears down the ones whose
# PR is closed. Open PRs keep their branch and deploy request.
set -euo pipefail

: "${PSCALE_ORG:?PSCALE_ORG is required}"
: "${PSCALE_DB:?PSCALE_DB is required}"
: "${GITHUB_REPOSITORY:?GITHUB_REPOSITORY is required}"
: "${GITHUB_TOKEN:?GITHUB_TOKEN is required}"

# Keep only branches older than a grace period so an in-flight job_pscale_pr
# branch is never deleted mid-run.
GRACE_SECONDS="${GRACE_SECONDS:-3600}"
export GRACE_SECONDS

pr_state() {
  # Prints "closed" when the PR for a pr-* branch is closed, "open" otherwise,
  # and returns non-zero when the branch is not a PR branch or has no PR.
  local branch="$1"
  local number="${branch#pr-}"
  case "$number" in
    *[!0-9]*) return 3 ;;
  esac
  curl -fsS --fail-with-body -H "Authorization: Bearer ${GITHUB_TOKEN}" \
    -H "Accept: application/vnd.github+json" \
    "https://api.github.com/repos/${GITHUB_REPOSITORY}/pulls/${number}" \
    | jq -r 'if .state == "closed" then "closed" else "open" end' 2>/dev/null || return 3
}

mapfile -t BRANCHES < <(
  mise exec -- pscale branch list "$PSCALE_DB" --org "$PSCALE_ORG" --format json \
    | mise exec -- jq -r '
        .[]
        | select(.name | startswith("pr-"))
        | select(
            (try (.created_at | sub("\\.[0-9]+Z$"; "Z") | fromdateiso8601) catch now)
            < (now - (env.GRACE_SECONDS | tonumber))
          )
        | .name
      '
)

echo "Found ${#BRANCHES[@]} pr-* branch(es) older than ${GRACE_SECONDS}s."

for branch in "${BRANCHES[@]}"; do
  case "$(pr_state "$branch")" in
    open)
      echo "Skipping $branch: PR still open."
      continue
      ;;
    closed)
      echo "Closing deploy requests and deleting $branch."
      mise exec -- pscale deploy-request list "$PSCALE_DB" --org "$PSCALE_ORG" --format json \
        | mise exec -- jq -r --arg b "$branch" '.[] | select(.branch==$b and .state=="open") | .number' \
        | while read -r num; do
            mise exec -- pscale deploy-request close "$PSCALE_DB" "$num" --org "$PSCALE_ORG" || true
          done
      mise exec -- pscale branch delete "$PSCALE_DB" "$branch" --org "$PSCALE_ORG" --force || true
      ;;
    *)
      echo "Skipping $branch: no PR or PR state unknown."
      continue
      ;;
  esac
done

echo "PlanetScale pr-* sweep complete."